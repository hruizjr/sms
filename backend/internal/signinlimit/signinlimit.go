// Package signinlimit is the layered, per-source limit of client portal sign-in attempts (Phase 4
// Group 6; owner decisions in decision log §23; security baseline §3 and §7).
//
// It is separate from, and does not change, the account lockout (five consecutive failed sign-ins
// lock an account for 15 minutes, owner decision G3-3): the lockout protects one account from many
// sources, these limits protect all accounts from one source. A refused attempt is refused before its
// credentials are looked at, so a refusal reveals nothing about any account and never counts toward,
// starts or extends an account lock.
//
// # Layers
//
// Every attempt passes two layers, in this order:
//
//  1. An all-attempt token bucket per source (IPv4 /32 or IPv6 /64): Config.RequestsPerSecond
//     tokens per second, up to Config.RequestBurst. Every attempt takes a token, whatever its
//     outcome. It bounds the request rate (and the password-hashing work) one source can cause.
//  2. Failed-attempt budgets: at most Config.FailuresPerIPv4 failures per IPv4 address, at most
//     Config.FailuresPerIPv6Subnet per IPv6 /64, and at most Config.FailuresPerIPv6Aggregate per IPv6
//     /48, within Config.FailureWindow. Before its credentials are checked, an attempt reserves one
//     slot in each budget of its source (the /32; or the /64 and its /48), all or none. A successful
//     sign-in, and an attempt whose password was never checked (the password-hashing capacity was
//     exhausted), returns exactly its own slots (Refund): earlier failures stay counted. So the users
//     of a shared address (an office NAT, carrier-grade NAT) who sign in successfully use no budget,
//     and an attacker gains nothing from a valid account of their own. Rotating IPv6 addresses inside
//     a /48 does not escape the /48 aggregate.
//
// # Shared and atomic
//
// Both layers live in PostgreSQL (portal_sign_in_request_buckets, portal_sign_in_failure_budgets), so
// every API instance shares them and a restart resets nothing. A token is taken by one atomic
// statement on the bucket's row. The budgets of an attempt are reserved in one transaction, always in
// the same order (the /48 before the /64), each by one atomic upsert that serializes concurrent
// attempts on its row; when any budget would exceed its limit the transaction is rolled back, so a
// refused attempt reserves nothing and concurrent attempts, on one instance or several, can never take
// more slots than the limit.
//
// # Sources
//
// The address is the one the HTTP layer determined (the direct peer, or the client named by a
// configured trusted proxy); this package never reads request headers. An invalid address (none
// could be determined) maps to the single source ::/128, shared by all such requests, with the IPv4
// limit.
//
// # Storage
//
// One bucket row per source seen within the time a bucket needs to refill, and one budget row per
// source with reserved slots in an unexpired window. Run deletes the rest periodically, so the tables
// hold at most the sources seen during one window (budgets) or one refill time (buckets) plus one
// cleanup interval.
package signinlimit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
)

// Config is the layered limit. Every field must be positive.
type Config struct {
	// FailuresPerIPv4 bounds the failed (or pending) attempts of one IPv4 address per FailureWindow.
	FailuresPerIPv4 int
	// FailuresPerIPv6Subnet bounds the failed attempts of one IPv6 /64 per FailureWindow.
	FailuresPerIPv6Subnet int
	// FailuresPerIPv6Aggregate bounds the failed attempts of one IPv6 /48 per FailureWindow.
	FailuresPerIPv6Aggregate int
	// FailureWindow is the fixed window of the failure budgets; it starts with a source's first
	// reservation.
	FailureWindow time.Duration
	// RequestsPerSecond is the refill rate of the all-attempt token bucket of one source.
	RequestsPerSecond float64
	// RequestBurst is the capacity of that bucket (the attempts a source can make at once).
	RequestBurst int
}

// Initial, configurable defaults (owner decision, decision log §23; backend/README.md).
const (
	DefaultFailuresPerIPv4          = 100
	DefaultFailuresPerIPv6Subnet    = 100
	DefaultFailuresPerIPv6Aggregate = 500
	DefaultFailureWindow            = 15 * time.Minute
	DefaultRequestsPerSecond        = 10
	DefaultRequestBurst             = 100
)

// DefaultConfig is the configuration with every default.
var DefaultConfig = Config{
	FailuresPerIPv4: DefaultFailuresPerIPv4, FailuresPerIPv6Subnet: DefaultFailuresPerIPv6Subnet,
	FailuresPerIPv6Aggregate: DefaultFailuresPerIPv6Aggregate, FailureWindow: DefaultFailureWindow,
	RequestsPerSecond: DefaultRequestsPerSecond, RequestBurst: DefaultRequestBurst,
}

// CleanupInterval is how often Run deletes ended windows and full buckets.
const CleanupInterval = time.Minute

// ReportInterval is the least time between two platform events recording the refusals of one
// source by the token bucket.
const ReportInterval = 5 * time.Minute

// maxBurst bounds RequestBurst (the database checks tokens <= 1,000,000).
const maxBurst = 1_000_000

// Layer names the limit that refused an attempt.
type Layer string

// Layers.
const (
	LayerRequestRate   Layer = "request_rate"   // the all-attempt token bucket
	LayerFailureBudget Layer = "failure_budget" // a failed-attempt budget
)

// Slot is one reserved failed-attempt slot: the budget's source and the window it belongs to.
type Slot struct {
	Source      netip.Prefix
	WindowStart time.Time
}

// Reservation is the slots an admitted attempt reserved; pass it to Refund after a successful sign-in.
type Reservation struct {
	Slots []Slot
}

// Refusal describes why an attempt was refused.
type Refusal struct {
	Layer Layer
	// Source is the bucket or budget that refused (for example an IPv6 /48).
	Source netip.Prefix
	// Limit is the budget's limit (failure budgets) or the bucket's burst (token bucket).
	Limit int
	// RetryAfter is how long until an attempt from the source can be admitted again (at least 1s).
	RetryAfter time.Duration
	// Report is true for the one refusal that should be recorded as a platform event: the first of
	// a budget's window, or the first of a bucket's ReportInterval. Later refusals are not recorded
	// one by one, so a flood cannot multiply audit writes.
	Report bool
}

// Admission is the result of Admit: either a Reservation (admitted) or a Refusal.
type Admission struct {
	Allowed     bool
	Reservation Reservation
	Refusal     Refusal
}

// Limiter applies the layered limit. It is safe for concurrent use.
type Limiter struct {
	db  *database.DB
	cfg Config
	now func() time.Time
}

// Option configures a Limiter.
type Option func(*Limiter)

// WithClock sets the clock (tests).
func WithClock(now func() time.Time) Option { return func(l *Limiter) { l.now = now } }

// New returns a Limiter. Every field of cfg must be positive.
func New(db *database.DB, cfg Config, opts ...Option) (*Limiter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	l := &Limiter{db: db, cfg: cfg, now: time.Now}
	for _, o := range opts {
		o(l)
	}
	return l, nil
}

// Validate reports an unusable configuration.
func (c Config) Validate() error {
	for _, n := range []int{c.FailuresPerIPv4, c.FailuresPerIPv6Subnet, c.FailuresPerIPv6Aggregate} {
		if n < 1 || n >= math.MaxInt32 {
			return fmt.Errorf("signinlimit: invalid failure limit %d", n)
		}
	}
	switch {
	case c.FailureWindow <= 0:
		return fmt.Errorf("signinlimit: invalid failure window %s", c.FailureWindow)
	case !(c.RequestsPerSecond > 0) || math.IsInf(c.RequestsPerSecond, 0):
		return fmt.Errorf("signinlimit: invalid request rate %v", c.RequestsPerSecond)
	case c.RequestBurst < 1 || c.RequestBurst > maxBurst:
		return fmt.Errorf("signinlimit: invalid request burst %d", c.RequestBurst)
	}
	return nil
}

// Config returns the limit.
func (l *Limiter) Config() Config { return l.cfg }

// Source returns the token-bucket source of a client address: the address itself for IPv4
// (including IPv4-mapped IPv6), its /64 prefix for IPv6, and ::/128 for an invalid address.
func Source(ip netip.Addr) netip.Prefix {
	if !ip.IsValid() {
		return netip.PrefixFrom(netip.IPv6Unspecified(), 128)
	}
	ip = ip.Unmap().WithZone("")
	if ip.Is4() {
		return netip.PrefixFrom(ip, 32)
	}
	p, _ := ip.Prefix(64) // cannot fail for a valid IPv6 address
	return p
}

// budget is one failure budget of an attempt.
type budget struct {
	source netip.Prefix
	limit  int
}

// budgets returns the failure budgets of ip in locking order (broadest first): the /32 for IPv4;
// the /48 and then the /64 for IPv6; ::/128 for an invalid address.
func (l *Limiter) budgets(ip netip.Addr) []budget {
	src := Source(ip)
	if !ip.IsValid() || src.Addr().Is4() {
		return []budget{{src, l.cfg.FailuresPerIPv4}}
	}
	aggregate, _ := src.Addr().Prefix(48) // cannot fail for a valid IPv6 address
	return []budget{{aggregate, l.cfg.FailuresPerIPv6Aggregate}, {src, l.cfg.FailuresPerIPv6Subnet}}
}

// errBudgetExceeded rolls back a reservation transaction.
var errBudgetExceeded = errors.New("failure budget exceeded")

// Admit applies both layers to one sign-in attempt from ip. An admitted attempt has taken a token
// and reserved its failure slots; call Refund with its Reservation if the sign-in succeeds or its
// password is never checked.
func (l *Limiter) Admit(ctx context.Context, ip netip.Addr) (Admission, error) {
	now := l.now()
	src := Source(ip)
	tok, err := sqlcgen.New(l.db.Pool()).TakeSignInRequestToken(ctx, sqlcgen.TakeSignInRequestTokenParams{
		Source: src, Burst: float64(l.cfg.RequestBurst), Now: now, Rate: l.cfg.RequestsPerSecond,
		ReportIntervalBefore: now.Add(-ReportInterval),
	})
	if err != nil {
		return Admission{}, fmt.Errorf("take sign-in request token: %w", err)
	}
	if !tok.LastAllowed {
		wait := time.Duration((1 - tok.Tokens) / l.cfg.RequestsPerSecond * float64(time.Second))
		return Admission{Refusal: Refusal{
			Layer: LayerRequestRate, Source: src, Limit: l.cfg.RequestBurst, RetryAfter: wholeSeconds(wait), Report: tok.LastReported,
		}}, nil
	}

	var (
		res     Reservation
		refused budget
		ended   time.Time // end of the refused budget's window
		started time.Time // start of the refused budget's window
	)
	err = l.db.InTx(ctx, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		for _, b := range l.budgets(ip) {
			row, err := q.ReserveSignInFailure(ctx, sqlcgen.ReserveSignInFailureParams{
				Source: b.source, Now: now, WindowEndedBefore: now.Add(-l.cfg.FailureWindow),
			})
			if err != nil {
				return err
			}
			if int(row.FailureCount) > b.limit {
				refused, started, ended = b, row.WindowStartAt, row.WindowStartAt.Add(l.cfg.FailureWindow)
				return errBudgetExceeded // rolls back every reservation of this attempt
			}
			res.Slots = append(res.Slots, Slot{Source: b.source, WindowStart: row.WindowStartAt})
		}
		return nil
	})
	switch {
	case errors.Is(err, errBudgetExceeded):
		n, err := sqlcgen.New(l.db.Pool()).MarkSignInFailureRefusalReported(ctx, sqlcgen.MarkSignInFailureRefusalReportedParams{
			Source: refused.source, WindowStartAt: started,
		})
		if err != nil {
			return Admission{}, fmt.Errorf("mark sign-in refusal: %w", err)
		}
		return Admission{Refusal: Refusal{
			Layer: LayerFailureBudget, Source: refused.source, Limit: refused.limit, RetryAfter: wholeSeconds(ended.Sub(now)), Report: n == 1,
		}}, nil
	case err != nil:
		return Admission{}, fmt.Errorf("reserve sign-in failure slots: %w", err)
	}
	return Admission{Allowed: true, Reservation: res}, nil
}

// Refund returns the slots of res. A slot whose window has ended since it was reserved is not
// returned (its window's count is gone anyway).
func (l *Limiter) Refund(ctx context.Context, res Reservation) error {
	q := sqlcgen.New(l.db.Pool())
	for _, s := range res.Slots {
		if _, err := q.RefundSignInFailure(ctx, sqlcgen.RefundSignInFailureParams{Source: s.Source, WindowStartAt: s.WindowStart}); err != nil {
			return fmt.Errorf("refund sign-in failure slot: %w", err)
		}
	}
	return nil
}

// wholeSeconds rounds d up to whole seconds, at least 1s, so that a client waiting exactly
// Retry-After is admitted.
func wholeSeconds(d time.Duration) time.Duration {
	return time.Duration(max(int64(math.Ceil(d.Seconds())), 1)) * time.Second
}

// Cleanup deletes ended or empty failure budgets and full token buckets, and returns how many rows
// it deleted.
func (l *Limiter) Cleanup(ctx context.Context) (int64, error) {
	now := l.now()
	q := sqlcgen.New(l.db.Pool())
	budgets, err := q.DeleteEndedSignInFailureBudgets(ctx, now.Add(-l.cfg.FailureWindow))
	if err != nil {
		return 0, fmt.Errorf("delete ended sign-in failure budgets: %w", err)
	}
	refill := time.Duration(float64(l.cfg.RequestBurst) / l.cfg.RequestsPerSecond * float64(time.Second))
	buckets, err := q.DeleteFullSignInRequestBuckets(ctx, sqlcgen.DeleteFullSignInRequestBucketsParams{
		FullBefore: now.Add(-refill), ReportIntervalBefore: now.Add(-ReportInterval),
	})
	if err != nil {
		return budgets, fmt.Errorf("delete full sign-in request buckets: %w", err)
	}
	return budgets + buckets, nil
}

// Run calls Cleanup every CleanupInterval until ctx is done. Failures are logged and retried at the
// next interval. Every API instance may run it; concurrent cleanups are harmless.
func (l *Limiter) Run(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cctx, cancel := context.WithTimeout(ctx, time.Minute)
			n, err := l.Cleanup(cctx)
			cancel()
			switch {
			case errors.Is(err, context.Canceled) && ctx.Err() != nil:
				return
			case err != nil:
				logger.Warn("sign-in limit cleanup failed", slog.String("error", err.Error()))
			case n > 0:
				logger.Debug("sign-in limit cleanup", slog.Int64("deleted", n))
			}
		}
	}
}
