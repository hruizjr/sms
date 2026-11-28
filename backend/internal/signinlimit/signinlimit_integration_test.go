//go:build integration

package signinlimit

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
)

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func open(t *testing.T, ctx context.Context, cfg config.DatabaseConfig) *database.DB {
	t.Helper()
	db, err := database.Open(ctx, cfg, "sms-signinlimit-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

func newLimiter(t *testing.T, db *database.DB, c *clock, cfg Config) *Limiter {
	t.Helper()
	l, err := New(db, cfg, WithClock(c.Now))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func attempt(t *testing.T, ctx context.Context, l *Limiter, ip string) Decision {
	t.Helper()
	d, err := l.Attempt(ctx, netip.MustParseAddr(ip))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// The limit per source, the window, Retry-After, the first refusal, independent sources, IPv6 /64
// grouping, separate instances sharing the database, a restart, and cleanup — as sms_app.
func TestIntegrationLimiter(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	c := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	cfg := Config{MaxAttempts: 3, Window: 10 * time.Minute}

	// Two API instances: separate limiters on separate connection pools, one database.
	instance1 := newLimiter(t, open(t, ctx, roles.App), c, cfg)
	instance2 := newLimiter(t, open(t, ctx, roles.App), c, cfg)

	// 1. The same source, alternating between instances: three attempts allowed, then refused.
	for i, l := range []*Limiter{instance1, instance2, instance1} {
		if d := attempt(t, ctx, l, "203.0.113.7"); !d.Allowed {
			t.Fatalf("attempt %d refused: %+v", i+1, d)
		}
	}
	c.Advance(90 * time.Second)
	d := attempt(t, ctx, instance2, "203.0.113.7")
	if d.Allowed || !d.FirstRefusal || d.RetryAfter != 510*time.Second {
		t.Errorf("4th attempt = %+v, want refused, first refusal, retry after 510s (window ends at 09:10)", d)
	}
	c.Advance(500*time.Millisecond + 9*time.Second)
	d = attempt(t, ctx, instance1, "203.0.113.7")
	if d.Allowed || d.FirstRefusal || d.RetryAfter != 501*time.Second {
		t.Errorf("5th attempt = %+v, want refused, not the first refusal, retry after 501s (rounded up)", d)
	}

	// 2. Other sources are independent; addresses of one IPv6 /64 are one source.
	if d := attempt(t, ctx, instance1, "198.51.100.20"); !d.Allowed {
		t.Errorf("another IPv4 source refused: %+v", d)
	}
	for i := range 3 {
		if d := attempt(t, ctx, instance1, []string{"2001:db8:1:2::1", "2001:db8:1:2::2", "2001:db8:1:2:ffff::9"}[i]); !d.Allowed {
			t.Errorf("IPv6 attempt %d refused: %+v", i+1, d)
		}
	}
	if d := attempt(t, ctx, instance2, "2001:db8:1:2::77"); d.Allowed {
		t.Errorf("4th attempt from the same /64 allowed")
	}
	if d := attempt(t, ctx, instance2, "2001:db8:1:3::1"); !d.Allowed {
		t.Errorf("another /64 refused: %+v", d)
	}

	// 3. A restart (a new limiter on a new pool) does not reset the count.
	restarted := newLimiter(t, open(t, ctx, roles.App), c, cfg)
	if d := attempt(t, ctx, restarted, "203.0.113.7"); d.Allowed {
		t.Errorf("attempt after restart allowed: the count was reset")
	}

	// 4. When the window has ended, the source starts again with a new window.
	c.Advance(10 * time.Minute)
	if d := attempt(t, ctx, restarted, "203.0.113.7"); !d.Allowed {
		t.Errorf("attempt after the window refused: %+v", d)
	}
	for range 2 {
		attempt(t, ctx, restarted, "203.0.113.7")
	}
	if d := attempt(t, ctx, restarted, "203.0.113.7"); d.Allowed || !d.FirstRefusal || d.RetryAfter != 10*time.Minute {
		t.Errorf("new window: 4th attempt = %+v, want the first refusal of the new window, retry after 10m", d)
	}

	// 5. Cleanup deletes only the windows that have ended (here: every source except 203.0.113.7).
	n, err := restarted.Cleanup(ctx)
	if err != nil || n != 3 {
		t.Errorf("Cleanup() = %d, %v; want 3 ended windows", n, err)
	}
	if d := attempt(t, ctx, restarted, "203.0.113.7"); d.Allowed {
		t.Errorf("cleanup removed an active window")
	}
}

// Concurrent attempts from one source, spread over several instances, cannot exceed the limit: the
// count is one atomic statement per attempt.
func TestIntegrationLimiterConcurrent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	c := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	cfg := Config{MaxAttempts: 10, Window: time.Hour}
	instances := []*Limiter{
		newLimiter(t, open(t, ctx, roles.App), c, cfg),
		newLimiter(t, open(t, ctx, roles.App), c, cfg),
		newLimiter(t, open(t, ctx, roles.App), c, cfg),
	}
	var allowed, firstRefusals atomic.Int32
	var wg sync.WaitGroup
	for i := range 60 {
		wg.Go(func() {
			d, err := instances[i%len(instances)].Attempt(ctx, netip.MustParseAddr("192.0.2.44"))
			if err != nil {
				t.Error(err)
				return
			}
			if d.Allowed {
				allowed.Add(1)
			}
			if d.FirstRefusal {
				firstRefusals.Add(1)
			}
		})
	}
	wg.Wait()
	if allowed.Load() != 10 || firstRefusals.Load() != 1 {
		t.Errorf("60 concurrent attempts: %d allowed, %d first refusals; want 10 and 1", allowed.Load(), firstRefusals.Load())
	}
	var count int
	if err := instances[0].db.Pool().QueryRow(ctx, `SELECT attempt_count FROM portal_sign_in_rate_limits WHERE source = '192.0.2.44/32'`).Scan(&count); err != nil || count != 60 {
		t.Errorf("stored count = %d, %v; want 60", count, err)
	}
}
