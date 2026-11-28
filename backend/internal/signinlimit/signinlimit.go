// Package signinlimit is the per-source rate limit of client portal sign-in attempts (Phase 4
// Group 6; security baseline §3 and §7).
//
// It is separate from, and does not change, the account lockout (five consecutive failed sign-ins
// lock an account for 15 minutes, owner decision G3-3): the lockout protects one account from many
// sources, this limit protects all accounts from one source. A limited source is refused before its
// credentials are looked at, so a refusal reveals nothing about any account and never counts toward,
// starts or extends an account lock.
//
// # Counting
//
// Each source has one fixed window that starts with its first attempt and lasts Config.Window. Every
// attempt in the window is counted, including refused ones; an attempt beyond Config.MaxAttempts is
// refused until the window ends. The count lives in PostgreSQL (portal_sign_in_rate_limits), so every
// API instance shares it and a restart does not reset it. Each attempt is counted by one atomic
// statement (INSERT ... ON CONFLICT DO UPDATE ... RETURNING), so concurrent attempts, on one instance
// or several, can never exceed the limit or be lost.
//
// # Sources
//
// A source is an IPv4 address or an IPv6 /64 prefix (one IPv6 subscriber or host usually controls a
// whole /64). The address is the one the HTTP layer determined (the direct peer, or the client named by
// a configured trusted proxy); this package never reads request headers.
//
// # Storage
//
// One row per source with an unexpired window. Cleanup deletes rows whose window has ended; Run does
// that periodically, so the table holds at most the sources seen during one window plus one cleanup
// interval.
package signinlimit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/netip"
	"time"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
)

// Config is the limit: at most MaxAttempts sign-in attempts per source within Window.
type Config struct {
	MaxAttempts int
	Window      time.Duration
}

// Defaults (documented in backend/README.md): generous enough that the users of one office behind a
// shared address can mistype passwords without being blocked, low enough to slow password guessing
// across many accounts from one source. The window matches the account lock duration (G3-3).
const (
	DefaultMaxAttempts = 50
	DefaultWindow      = 15 * time.Minute
)

// CleanupInterval is how often Run deletes ended windows.
const CleanupInterval = 5 * time.Minute

// Decision is the result of counting one attempt.
type Decision struct {
	// Allowed is false when the attempt exceeds the limit and must be refused.
	Allowed bool
	// RetryAfter is how long a refused source must wait until its window ends (at least 1s).
	RetryAfter time.Duration
	// FirstRefusal is true for the first refused attempt of a window, so that a refusal can be
	// recorded once per source and window rather than once per request.
	FirstRefusal bool
}

// Limiter counts sign-in attempts. It is safe for concurrent use.
type Limiter struct {
	db  *database.DB
	cfg Config
	now func() time.Time
}

// Option configures a Limiter.
type Option func(*Limiter)

// WithClock sets the clock (tests).
func WithClock(now func() time.Time) Option { return func(l *Limiter) { l.now = now } }

// New returns a Limiter. cfg must have a positive MaxAttempts and Window.
func New(db *database.DB, cfg Config, opts ...Option) (*Limiter, error) {
	if cfg.MaxAttempts < 1 || cfg.MaxAttempts >= math.MaxInt32 || cfg.Window <= 0 {
		return nil, fmt.Errorf("signinlimit: invalid limit %d per %s", cfg.MaxAttempts, cfg.Window)
	}
	l := &Limiter{db: db, cfg: cfg, now: time.Now}
	for _, o := range opts {
		o(l)
	}
	return l, nil
}

// Config returns the limit.
func (l *Limiter) Config() Config { return l.cfg }

// Source returns the rate-limit source of a client address: the address itself for IPv4 (including
// IPv4-mapped IPv6), its /64 prefix for IPv6. An invalid address (no address could be determined)
// maps to the single source ::/128, shared by all such requests.
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

// Attempt counts one sign-in attempt from ip and decides whether it may proceed.
func (l *Limiter) Attempt(ctx context.Context, ip netip.Addr) (Decision, error) {
	now := l.now()
	row, err := sqlcgen.New(l.db.Pool()).CountSignInAttempt(ctx, sqlcgen.CountSignInAttemptParams{
		Source: Source(ip), Now: now, WindowEndedBefore: now.Add(-l.cfg.Window),
	})
	if err != nil {
		return Decision{}, fmt.Errorf("count sign-in attempt: %w", err)
	}
	count := int(row.AttemptCount)
	if count <= l.cfg.MaxAttempts {
		return Decision{Allowed: true}, nil
	}
	// Whole seconds, rounded up, so that a client waiting exactly Retry-After finds the window ended.
	wait := row.WindowStartAt.Add(l.cfg.Window).Sub(now)
	seconds := max(int64(math.Ceil(wait.Seconds())), 1)
	return Decision{
		RetryAfter:   time.Duration(seconds) * time.Second,
		FirstRefusal: count == l.cfg.MaxAttempts+1,
	}, nil
}

// Cleanup deletes the counters of windows that have ended and returns how many it deleted.
func (l *Limiter) Cleanup(ctx context.Context) (int64, error) {
	n, err := sqlcgen.New(l.db.Pool()).DeleteEndedSignInWindows(ctx, l.now().Add(-l.cfg.Window))
	if err != nil {
		return 0, fmt.Errorf("delete ended sign-in windows: %w", err)
	}
	return n, nil
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
				logger.Warn("sign-in rate limit cleanup failed", slog.String("error", err.Error()))
			case n > 0:
				logger.Debug("sign-in rate limit cleanup", slog.Int64("deleted", n))
			}
		}
	}
}
