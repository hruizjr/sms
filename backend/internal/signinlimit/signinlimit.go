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
// # Storage and cleanup
//
// One bucket row per source seen within the time a bucket needs to refill, and one budget row per
// source that reserved a slot in an unexpired window. Run deletes the rest: one cycle every
// CleanupInterval, each cycle a sequence of bounded batches (CleanupBatchSize rows, each batch its own
// transaction) up to CleanupMaxBatches per table, so one cycle's work is bounded however many rows are
// waiting. A batch locks its rows with FOR UPDATE SKIP LOCKED: cleanup never waits for a row that a
// sign-in, or another instance's cleanup, holds, so it cannot deadlock with a sign-in (which locks an
// IPv6 /48 before its /64) and several instances share the work instead of contending for it.
// Correctness never depends on cleanup: an ended window restarts on its next reservation and a bucket
// refills by computation, so a failed or skipped cleanup only leaves rows in place.
//
// The tables are bounded by the number of distinct sources seen per window, not by a fixed number: an
// attacker who can send from many addresses creates one row per new address. Cleanup removes at most
// CleanupBatchSize × CleanupMaxBatches rows per table per cycle and instance; a cycle that stops at
// that bound is reported as incomplete (CleanupResult.Complete), which monitoring must alert on, and
// edge rate limiting must bound the rate of new sources (security baseline S-12).

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

// Cleanup bounds. CleanupInterval is how often Run starts a cleanup cycle; a cycle deletes at most
// CleanupMaxBatches batches of CleanupBatchSize rows from each table (50,000 rows per table, cycle and
// instance), each batch bounded by cleanupBatchTimeout, and the cycle by cleanupCycleTimeout.
const (
	CleanupInterval     = time.Minute
	CleanupBatchSize    = 1000
	CleanupMaxBatches   = 50
	cleanupBatchTimeout = 10 * time.Second
	cleanupCycleTimeout = time.Minute
)

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
	db      *database.DB
	cfg     Config
	now     func() time.Time
	metrics CleanupMetrics
	batch   int32 // rows per cleanup batch
	batches int   // batches per table and cycle
}

// CleanupMetrics receives the outcome of every cleanup cycle run by Run (implemented by
// observability.Metrics). succeeded is false when a batch failed; complete is false when the cycle
// stopped at its work bound with rows possibly left; the deleted counts include the batches committed
// before a failure.
type CleanupMetrics interface {
	ObserveSignInLimitCleanup(succeeded, complete bool, duration time.Duration, failureBudgetsDeleted, requestBucketsDeleted int64)
}

// Option configures a Limiter.
type Option func(*Limiter)

// WithClock sets the clock (tests).
func WithClock(now func() time.Time) Option { return func(l *Limiter) { l.now = now } }

// WithCleanupMetrics reports every cleanup cycle of Run to m.
func WithCleanupMetrics(m CleanupMetrics) Option { return func(l *Limiter) { l.metrics = m } }

// WithCleanupBounds replaces CleanupBatchSize and CleanupMaxBatches (tests).
func WithCleanupBounds(batchSize, maxBatches int) Option {
	return func(l *Limiter) { l.batch, l.batches = int32(min(max(batchSize, 1), 1_000_000)), max(maxBatches, 1) } //nolint:gosec // bounded above
}

// New returns a Limiter. Every field of cfg must be positive.
func New(db *database.DB, cfg Config, opts ...Option) (*Limiter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	l := &Limiter{db: db, cfg: cfg, now: time.Now, batch: CleanupBatchSize, batches: CleanupMaxBatches}
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

// CleanupResult is what one cleanup cycle deleted.
type CleanupResult struct {
	FailureBudgets int64 // budgets whose window had ended
	RequestBuckets int64 // buckets that were full again
	// Complete is false when a table still had rows to delete after CleanupMaxBatches batches: the
	// cycle stopped at its work bound and the next cycle continues.
	Complete bool
}

// Cleanup runs one cleanup cycle: it deletes budgets whose window has ended and buckets that are
// full again, in batches of at most CleanupBatchSize rows that each commit on their own, at most
// CleanupMaxBatches per table. Rows locked by a sign-in or another cleanup are skipped, never waited
// for. The tables are cleaned independently (a failure on one does not skip the other); on error it
// returns what the committed batches deleted.
func (l *Limiter) Cleanup(ctx context.Context) (CleanupResult, error) {
	now := l.now()
	q := sqlcgen.New(l.db.Pool())
	var res CleanupResult
	budgets, budgetsComplete, budgetsErr := l.deleteBatches(ctx, func(ctx context.Context) (int64, error) {
		return q.DeleteEndedSignInFailureBudgets(ctx, sqlcgen.DeleteEndedSignInFailureBudgetsParams{
			WindowEndedBefore: now.Add(-l.cfg.FailureWindow), BatchSize: l.batch,
		})
	})
	refill := time.Duration(float64(l.cfg.RequestBurst) / l.cfg.RequestsPerSecond * float64(time.Second))
	buckets, bucketsComplete, bucketsErr := l.deleteBatches(ctx, func(ctx context.Context) (int64, error) {
		return q.DeleteFullSignInRequestBuckets(ctx, sqlcgen.DeleteFullSignInRequestBucketsParams{
			FullBefore: now.Add(-refill), ReportIntervalBefore: now.Add(-ReportInterval), BatchSize: l.batch,
		})
	})
	res = CleanupResult{FailureBudgets: budgets, RequestBuckets: buckets, Complete: budgetsComplete && bucketsComplete}
	var errs []error
	if budgetsErr != nil {
		errs = append(errs, fmt.Errorf("delete ended sign-in failure budgets: %w", budgetsErr))
	}
	if bucketsErr != nil {
		errs = append(errs, fmt.Errorf("delete full sign-in request buckets: %w", bucketsErr))
	}
	return res, errors.Join(errs...)
}

// deleteBatches runs batch until it deletes fewer than a full batch (complete) or l.batches batches
// have run (not complete), and returns the rows deleted.
func (l *Limiter) deleteBatches(ctx context.Context, batch func(context.Context) (int64, error)) (deleted int64, complete bool, err error) {
	for range l.batches {
		bctx, cancel := context.WithTimeout(ctx, cleanupBatchTimeout)
		n, err := batch(bctx)
		cancel()
		if err != nil {
			return deleted, false, err
		}
		deleted += n
		if n < int64(l.batch) {
			return deleted, true, nil
		}
	}
	return deleted, false, nil
}

// Run runs a cleanup cycle at once and then every CleanupInterval until ctx is done, reporting each
// cycle to the CleanupMetrics (WithCleanupMetrics). A failed cycle is logged as an error and counted
// as a failure; the next cycle tries again. Every API instance runs it; their batches skip each
// other's rows.
func (l *Limiter) Run(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	for {
		if !l.cleanupCycle(ctx, logger) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// cleanupCycle runs and reports one cycle; it returns false when ctx ended.
func (l *Limiter) cleanupCycle(ctx context.Context, logger *slog.Logger) bool {
	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, cleanupCycleTimeout)
	res, err := l.Cleanup(cctx)
	cancel()
	if err != nil && ctx.Err() != nil {
		return false // shutting down: not a cleanup failure
	}
	if l.metrics != nil {
		l.metrics.ObserveSignInLimitCleanup(err == nil, res.Complete, time.Since(start), res.FailureBudgets, res.RequestBuckets)
	}
	attrs := []any{
		slog.Int64("failure_budgets_deleted", res.FailureBudgets), slog.Int64("request_buckets_deleted", res.RequestBuckets),
		slog.Bool("complete", res.Complete), slog.Duration("duration", time.Since(start)),
	}
	switch {
	case err != nil:
		logger.Error("sign-in limit cleanup failed", append(attrs, slog.String("error", err.Error()))...)
	case !res.Complete:
		logger.Warn("sign-in limit cleanup stopped at its work bound; rows remain", attrs...)
	case res.FailureBudgets+res.RequestBuckets > 0:
		logger.Debug("sign-in limit cleanup", attrs...)
	}
	return true
}
