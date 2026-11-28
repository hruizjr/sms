//go:build integration

package signinlimit

import (
	"context"
	"fmt"
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

// openAdmin opens the bootstrap superuser's pool (test assertions only; never the limiter's).
func openAdmin(t *testing.T, ctx context.Context, cfg config.DatabaseConfig) *database.DB {
	t.Helper()
	db, err := database.Open(ctx, cfg, "sms-signinlimit-test-admin")
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

func admit(t *testing.T, ctx context.Context, l *Limiter, ip string) Admission {
	t.Helper()
	a, err := l.Admit(ctx, netip.MustParseAddr(ip))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func refund(t *testing.T, ctx context.Context, l *Limiter, a Admission) {
	t.Helper()
	if err := l.Refund(ctx, a.Reservation); err != nil {
		t.Fatal(err)
	}
}

// A configuration whose token bucket never interferes (for the failure-budget tests).
func budgetsOnly(ipv4, ipv6Subnet, ipv6Aggregate int) Config {
	return Config{
		FailuresPerIPv4: ipv4, FailuresPerIPv6Subnet: ipv6Subnet, FailuresPerIPv6Aggregate: ipv6Aggregate,
		FailureWindow: 15 * time.Minute, RequestsPerSecond: 1000, RequestBurst: 1_000_000,
	}
}

func setup(t *testing.T) (context.Context, *database.DB, *database.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	return ctx, open(t, ctx, roles.App), openAdmin(t, ctx, roles.Admin)
}

func count(t *testing.T, ctx context.Context, admin *database.DB, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := admin.Pool().QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Failures are counted per source; a success refunds only its own slot, so earlier failures stay
// counted; the window, Retry-After, the one reported refusal per window, and a restart — as sms_app.
func TestIntegrationFailureBudgetReserveAndRefund(t *testing.T) {
	t.Parallel()
	ctx, db, admin := setup(t)
	c := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	l := newLimiter(t, db, c, budgetsOnly(5, 5, 20))
	const ip = "203.0.113.10"

	// Three failures, then a success: the success's slot is returned, the failures stay.
	for range 3 {
		if a := admit(t, ctx, l, ip); !a.Allowed {
			t.Fatal("failure refused under the limit")
		}
	}
	ok := admit(t, ctx, l, ip)
	if !ok.Allowed || len(ok.Reservation.Slots) != 1 {
		t.Fatalf("admission = %+v", ok)
	}
	refund(t, ctx, l, ok)
	if n := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '203.0.113.10/32'`); n != 3 {
		t.Fatalf("failure_count after 3 failures and 1 success = %d, want 3", n)
	}
	// Many successes from the same address (a NAT of legitimate users) never use the budget.
	for range 50 {
		a := admit(t, ctx, l, ip)
		if !a.Allowed {
			t.Fatal("a successful sign-in was refused")
		}
		refund(t, ctx, l, a)
	}
	// Two more failures reach the limit; the next attempt is refused and reported once.
	admit(t, ctx, l, ip)
	admit(t, ctx, l, ip)
	c.Advance(time.Minute)
	refused := admit(t, ctx, l, ip)
	if refused.Allowed || refused.Refusal.Layer != LayerFailureBudget || !refused.Refusal.Report ||
		refused.Refusal.Limit != 5 || refused.Refusal.Source != netip.MustParsePrefix("203.0.113.10/32") ||
		refused.Refusal.RetryAfter != 14*time.Minute {
		t.Fatalf("over the limit: %+v", refused)
	}
	if again := admit(t, ctx, l, ip); again.Allowed || again.Refusal.Report {
		t.Errorf("second refusal: %+v, want refused and not reported", again)
	}
	// A refused attempt reserved nothing; a refund of a refused admission is a no-op.
	refund(t, ctx, l, refused)
	if n := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '203.0.113.10/32'`); n != 5 {
		t.Errorf("failure_count = %d, want 5", n)
	}
	// Another address is independent.
	if a := admit(t, ctx, l, "203.0.113.11"); !a.Allowed {
		t.Error("another address was refused")
	}
	// A restart (a new limiter on the same database) keeps the count.
	if a := admit(t, ctx, newLimiter(t, db, c, budgetsOnly(5, 5, 20)), ip); a.Allowed {
		t.Error("a new limiter forgot the count")
	}
	// After the window: admitted again, and a refund of an old window's slot does not reduce the new one.
	c.Advance(14 * time.Minute)
	fresh := admit(t, ctx, l, ip)
	if !fresh.Allowed {
		t.Fatal("refused after the window ended")
	}
	if err := l.Refund(ctx, Reservation{Slots: []Slot{{Source: netip.MustParsePrefix("203.0.113.10/32"), WindowStart: c.Now().Add(-15 * time.Minute)}}}); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '203.0.113.10/32'`); n != 1 {
		t.Errorf("failure_count in the new window = %d, want 1", n)
	}
	// A new window may report its own refusal again.
	for range 4 {
		admit(t, ctx, l, ip)
	}
	if r := admit(t, ctx, l, ip); r.Allowed || !r.Refusal.Report {
		t.Errorf("first refusal of the new window: %+v, want reported", r)
	}

	// Cleanup removes ended windows only. A budget whose slots were all refunded stays until its
	// window ends (an empty budget behaves like an absent one).
	c.Advance(16 * time.Minute)
	admit(t, ctx, l, "203.0.113.12") // a pending slot in an open window
	empty := admit(t, ctx, l, "203.0.113.13")
	refund(t, ctx, l, empty)
	res, err := l.Cleanup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.FailureBudgets != 2 || !res.Complete { // the two ended windows of .10 and .11
		t.Errorf("cleanup = %+v, want 2 ended budgets deleted, complete", res)
	}
	if n := count(t, ctx, admin, `SELECT count(*) FROM portal_sign_in_failure_budgets`); n != 2 {
		t.Errorf("%d budgets after cleanup, want 2 (the open and the open empty one)", n)
	}
	// The empty budget keeps counting in its own window: a failure after the refund is counted.
	if a := admit(t, ctx, l, "203.0.113.13"); !a.Allowed {
		t.Fatal("empty budget refused")
	}
	if n := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '203.0.113.13/32'`); n != 1 {
		t.Errorf("failure_count of the empty budget after one failure = %d, want 1", n)
	}
	c.Advance(15 * time.Minute)
	if res, err := l.Cleanup(ctx); err != nil || res.FailureBudgets != 2 {
		t.Errorf("cleanup after the windows ended = %+v, %v; want 2 deleted", res, err)
	}
	if n := count(t, ctx, admin, `SELECT count(*) FROM portal_sign_in_failure_budgets`); n != 0 {
		t.Errorf("%d budgets after their windows ended, want 0", n)
	}
}

// IPv6: the /64 and the /48 aggregate are both reserved; rotating /64s inside one /48 is stopped
// by the aggregate; a refusal by the /48 reserves nothing in the /64.
func TestIntegrationIPv6AggregateStopsRotation(t *testing.T) {
	t.Parallel()
	ctx, db, admin := setup(t)
	c := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	l := newLimiter(t, db, c, budgetsOnly(100, 3, 10))

	// One /64: three failures, then refused by the /64 (other addresses in it are the same source).
	for i := range 3 {
		if a := admit(t, ctx, l, fmt.Sprintf("2001:db8:aa:1::%x", i+1)); !a.Allowed || len(a.Reservation.Slots) != 2 {
			t.Fatalf("attempt %d: %+v", i, a)
		}
	}
	if r := admit(t, ctx, l, "2001:db8:aa:1:ffff::1"); r.Allowed || r.Refusal.Source != netip.MustParsePrefix("2001:db8:aa:1::/64") {
		t.Fatalf("fourth attempt in the /64: %+v, want refused by the /64", r)
	}
	// Rotating to new /64s in the same /48: admitted until the /48 has 10 failures.
	admitted := 3
	var refusal Refusal
	for i := 2; i < 100; i++ {
		a := admit(t, ctx, l, fmt.Sprintf("2001:db8:aa:%x::1", i))
		if !a.Allowed {
			refusal = a.Refusal
			break
		}
		admitted++
	}
	if admitted != 10 || refusal.Source != netip.MustParsePrefix("2001:db8:aa::/48") || refusal.Limit != 10 || !refusal.Report {
		t.Fatalf("rotation: %d failures admitted, refusal %+v; want 10 and a reported /48 refusal", admitted, refusal)
	}
	// The /48 refusal rolled back: the new /64 holds no slot.
	if n := count(t, ctx, admin, `SELECT COALESCE(sum(failure_count), 0) FROM portal_sign_in_failure_budgets WHERE masklen(source) = 64`); n != 10 {
		t.Errorf("sum over the /64 budgets = %d, want 10", n)
	}
	if n := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '2001:db8:aa::/48'`); n != 10 {
		t.Errorf("/48 count = %d, want 10", n)
	}
	// Another /48 is independent.
	if a := admit(t, ctx, l, "2001:db8:bb:1::1"); !a.Allowed {
		t.Error("another /48 was refused")
	}
	// A success inside the full /48 is refused too (its slot cannot be reserved), until a refund or
	// the end of the window; a refund of an earlier success restores one slot.
	c.Advance(15 * time.Minute)
	ok := admit(t, ctx, l, "2001:db8:aa:50::1")
	refund(t, ctx, l, ok)
	if n := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '2001:db8:aa::/48'`); n != 0 {
		t.Errorf("/48 count after a refunded success in a new window = %d, want 0", n)
	}
}

// Concurrent attempts from one source on several API instances never reserve more slots than the
// limit, and concurrent successes refund exactly their own slots.
func TestIntegrationFailureBudgetConcurrent(t *testing.T) {
	t.Parallel()
	roles := dbtest.NewMigratedDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	admin := openAdmin(t, ctx, roles.Admin)
	c := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	cfg := budgetsOnly(25, 25, 40)
	instances := make([]*Limiter, 3)
	for i := range instances {
		instances[i] = newLimiter(t, open(t, ctx, roles.App), c, cfg) // separate pools, like separate instances
	}

	var allowed, refused, reports atomic.Int64
	var wg sync.WaitGroup
	for i := range 120 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := instances[i%len(instances)]
			ip := netip.MustParseAddr(fmt.Sprintf("2001:db8:cc:%x::1", i%6)) // 6 /64s of one /48
			a, err := l.Admit(ctx, ip)
			if err != nil {
				t.Error(err)
				return
			}
			if !a.Allowed {
				refused.Add(1)
				if a.Refusal.Report {
					reports.Add(1)
				}
				return
			}
			allowed.Add(1)
			if i%2 == 0 { // half of the admitted attempts succeed and refund
				if err := l.Refund(ctx, a.Reservation); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	// The /48 holds at most 40 slots at any time; refunds free some during the run, so more than 40
	// may be admitted overall, but the committed count never exceeds the limit.
	got := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '2001:db8:cc::/48'`)
	if got > 40 || got < 1 {
		t.Errorf("/48 count = %d, want 1..40", got)
	}
	sum64 := count(t, ctx, admin, `SELECT COALESCE(sum(failure_count), 0) FROM portal_sign_in_failure_budgets WHERE masklen(source) = 64`)
	if sum64 != got {
		t.Errorf("sum of the /64 counts = %d, /48 count = %d; every slot is reserved in both", sum64, got)
	}
	if allowed.Load()+refused.Load() != 120 {
		t.Errorf("allowed %d + refused %d != 120", allowed.Load(), refused.Load())
	}
	if refused.Load() > 0 && (reports.Load() < 1 || reports.Load() > 7) { // at most one per budget and window
		t.Errorf("%d refusals reported, want 1 to 7", reports.Load())
	}
	t.Logf("concurrent: %d admitted, %d refused, %d reports, final /48 count %d", allowed.Load(), refused.Load(), reports.Load(), got)
}

// The all-attempt token bucket: the burst, the refill rate, Retry-After, one report per interval,
// independent sources, shared by instances, safe with clocks that disagree, and cleanup.
func TestIntegrationRequestTokenBucket(t *testing.T) {
	t.Parallel()
	roles := dbtest.NewMigratedDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	db, admin := open(t, ctx, roles.App), openAdmin(t, ctx, roles.Admin)
	c := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)}
	cfg := Config{FailuresPerIPv4: 1_000_000, FailuresPerIPv6Subnet: 1_000_000, FailuresPerIPv6Aggregate: 1_000_000,
		FailureWindow: 15 * time.Minute, RequestsPerSecond: 10, RequestBurst: 100}
	l := newLimiter(t, db, c, cfg)
	other := newLimiter(t, open(t, ctx, roles.App), c, cfg) // a second instance with its own pool
	const ip = "198.51.100.20"

	for i := range 100 {
		inst := l
		if i%2 == 1 {
			inst = other // two instances share the bucket
		}
		if a := admit(t, ctx, inst, ip); !a.Allowed {
			t.Fatalf("request %d of the burst refused", i+1)
		}
	}
	r := admit(t, ctx, l, ip)
	if r.Allowed || r.Refusal.Layer != LayerRequestRate || !r.Refusal.Report || r.Refusal.RetryAfter != time.Second ||
		r.Refusal.Source != netip.MustParsePrefix("198.51.100.20/32") {
		t.Fatalf("request 101: %+v, want refused by the token bucket, reported, Retry-After 1s", r)
	}
	if again := admit(t, ctx, other, ip); again.Allowed || again.Refusal.Report {
		t.Errorf("request 102: %+v, want refused and not reported", again)
	}
	// Refused requests take no token and reserve no failure slot.
	if n := count(t, ctx, admin, `SELECT failure_count FROM portal_sign_in_failure_budgets WHERE source = '198.51.100.20/32'`); n != 100 {
		t.Errorf("failure slots = %d, want 100 (only admitted attempts)", n)
	}
	// 10 tokens per second: after 0.5 s five requests pass, the sixth is refused.
	c.Advance(500 * time.Millisecond)
	for i := range 5 {
		if a := admit(t, ctx, l, ip); !a.Allowed {
			t.Fatalf("refill request %d refused", i+1)
		}
	}
	if a := admit(t, ctx, l, ip); a.Allowed {
		t.Error("more requests than the refill rate allows")
	}
	// An instance whose clock is behind adds no tokens (time never runs backwards for a bucket).
	behind := newLimiter(t, db, &clock{now: c.Now().Add(-time.Hour)}, cfg)
	if a := admit(t, ctx, behind, ip); a.Allowed {
		t.Error("an instance with a slow clock got a token")
	}
	// Other sources are independent; an IPv6 /64 is one source.
	if a := admit(t, ctx, l, "198.51.100.21"); !a.Allowed {
		t.Error("another address was refused")
	}
	for i := range 100 {
		admit(t, ctx, l, fmt.Sprintf("2001:db8:dd:1::%x", i+1))
	}
	if a := admit(t, ctx, l, "2001:db8:dd:1:ffff::1"); a.Allowed || a.Refusal.Layer != LayerRequestRate {
		t.Errorf("101st request of one /64: %+v, want refused", a)
	}
	// A new report after the report interval, while still refused.
	c.Advance(ReportInterval)
	for range 100 {
		admit(t, ctx, l, ip) // the bucket refilled completely; empty it again
	}
	if a := admit(t, ctx, l, ip); a.Allowed || !a.Refusal.Report {
		t.Errorf("refusal after the report interval: %+v, want reported", a)
	}
	// Cleanup removes buckets that are full again and whose report is old.
	c.Advance(ReportInterval + 11*time.Second)
	if _, err := l.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, admin, `SELECT count(*) FROM portal_sign_in_request_buckets`); n != 0 {
		t.Errorf("%d buckets after cleanup, want 0", n)
	}
}

// The token bucket is exact under concurrency across instances: of N simultaneous requests from
// one source, exactly the burst is admitted.
func TestIntegrationRequestTokenBucketConcurrent(t *testing.T) {
	t.Parallel()
	roles := dbtest.NewMigratedDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	frozen := &clock{now: time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)} // no refill during the run
	cfg := Config{FailuresPerIPv4: 1_000_000, FailuresPerIPv6Subnet: 1_000_000, FailuresPerIPv6Aggregate: 1_000_000,
		FailureWindow: 15 * time.Minute, RequestsPerSecond: 10, RequestBurst: 30}
	instances := make([]*Limiter, 3)
	for i := range instances {
		instances[i] = newLimiter(t, open(t, ctx, roles.App), frozen, cfg)
	}
	var allowed, reports atomic.Int64
	var wg sync.WaitGroup
	for i := range 150 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, err := instances[i%3].Admit(ctx, netip.MustParseAddr("192.0.2.77"))
			if err != nil {
				t.Error(err)
				return
			}
			if a.Allowed {
				allowed.Add(1)
			} else if a.Refusal.Report {
				reports.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 30 || reports.Load() != 1 {
		t.Errorf("%d admitted and %d reports, want exactly 30 and 1", allowed.Load(), reports.Load())
	}
}

// sms_app has exactly the privileges the limiter needs on its tables, and no TRUNCATE.
func TestIntegrationLimiterPrivileges(t *testing.T) {
	t.Parallel()
	ctx, db, _ := setup(t)
	for _, sql := range []string{
		`TRUNCATE portal_sign_in_failure_budgets`,
		`TRUNCATE portal_sign_in_request_buckets`,
		`UPDATE portal_sign_in_failure_budgets SET source = '192.0.2.1/32'`,
		`UPDATE portal_sign_in_request_buckets SET source = '192.0.2.1/32'`,
		`INSERT INTO portal_sign_in_failure_budgets (source, window_start_at, failure_count, refusal_reported_for) VALUES ('192.0.2.1/32', now(), 1, now())`,
	} {
		if _, err := db.Pool().Exec(ctx, sql); err == nil {
			t.Errorf("as sms_app, %q succeeded", sql)
		}
	}
}
