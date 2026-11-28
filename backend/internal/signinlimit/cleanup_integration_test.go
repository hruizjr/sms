//go:build integration

package signinlimit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
)

// Regression tests of the cleanup lock interaction (final security review of PR #23). A sign-in
// reservation locks an IPv6 /48 budget before its /64 budget; the former cleanup was one DELETE that
// locked expired rows in table order (the /64 first when it is stored first) and could deadlock with
// it. Cleanup now deletes bounded batches chosen with FOR UPDATE SKIP LOCKED and never waits for a
// row a sign-in holds. The interaction is driven step by step: each step waits until PostgreSQL shows
// the expected lock wait (pg_stat_activity), not for a fixed time.

var t0 = time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)

// setupRoles is setup that also returns the roles, for tests that open one pool per instance.
func setupRoles(t *testing.T) (context.Context, dbtest.Roles, *database.DB, *database.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	return ctx, roles, open(t, ctx, roles.App), openAdmin(t, ctx, roles.Admin)
}

// seedBudget stores a budget row directly (as the bootstrap superuser), in the order of the calls, so
// the test controls the table order.
func seedBudget(t *testing.T, ctx context.Context, admin *database.DB, source string, windowStart time.Time, n int) {
	t.Helper()
	if _, err := admin.Pool().Exec(ctx, `INSERT INTO portal_sign_in_failure_budgets (source, window_start_at, failure_count) VALUES ($1, $2, $3)`,
		source, windowStart, n); err != nil {
		t.Fatal(err)
	}
}

// budgetRow returns a budget's count and window start (ok false when it does not exist).
func budgetRow(t *testing.T, ctx context.Context, admin *database.DB, source string) (n int, start time.Time, ok bool) {
	t.Helper()
	err := admin.Pool().QueryRow(ctx, `SELECT failure_count, window_start_at FROM portal_sign_in_failure_budgets WHERE source = $1`, source).Scan(&n, &start)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, time.Time{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return n, start, true
}

// backendPID returns the PostgreSQL backend of tx.
func backendPID(t *testing.T, ctx context.Context, tx pgx.Tx) int32 {
	t.Helper()
	var pid int32
	if err := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

// waitForLockWait waits until backend pid is blocked on a lock (deterministic synchronization).
func waitForLockWait(t *testing.T, ctx context.Context, admin *database.DB, pid int32) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := admin.Pool().QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock')`, pid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		time.Sleep(5 * time.Millisecond) // polling the catalog, not synchronizing on time
	}
	t.Fatalf("backend %d never waited for a lock", pid)
}

func reserve(ctx context.Context, tx pgx.Tx, source string, now time.Time) (sqlcgen.ReserveSignInFailureRow, error) {
	return sqlcgen.New(tx).ReserveSignInFailure(ctx, sqlcgen.ReserveSignInFailureParams{
		Source: netip.MustParsePrefix(source), Now: now, WindowEndedBefore: now.Add(-15 * time.Minute),
	})
}

const (
	aggregate48 = "2001:db8:42::/48"
	subnet64    = "2001:db8:42:1::/64"
)

// The original interaction, step by step: (1) a sign-in locks the /48; (2) a cleanup batch takes the
// /64 and skips the locked /48 without waiting; (3) the sign-in continues to its /64 and waits for the
// batch; (4) the batch commits, the sign-in completes and commits. No deadlock; the reservation is
// neither lost nor reset; the skipped row is left to a later cycle.
func TestIntegrationCleanupSkipsRowsLockedBySignIn(t *testing.T) {
	t.Parallel()
	ctx, db, admin := setup(t)
	ended := t0.Add(-20 * time.Minute)
	// The /64 is stored before its /48: the table order in which the old DELETE deadlocked.
	seedBudget(t, ctx, admin, subnet64, ended, 7)
	seedBudget(t, ctx, admin, aggregate48, ended, 7)
	seedBudget(t, ctx, admin, "198.51.100.1/32", ended, 3) // unrelated and ended

	// (1) The sign-in transaction reserves its /48 (the window had ended: it starts again at t0).
	signIn, err := db.Pool().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = signIn.Rollback(ctx) }() //nolint:errcheck // no-op after commit
	signInPID := backendPID(t, ctx, signIn)
	if row, err := reserve(ctx, signIn, aggregate48, t0); err != nil || row.FailureCount != 1 || !row.WindowStartAt.Equal(t0) {
		t.Fatalf("reserve /48 = %+v, %v; want a new window with 1 slot", row, err)
	}

	// (2) A cleanup batch, in its own transaction, deletes the ended /64 and the ended IPv4 budget and
	// skips the locked /48. It must not wait: a bounded context fails the test if it blocks.
	cleanup, err := db.Pool().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cleanup.Rollback(ctx) }() //nolint:errcheck // no-op after commit
	bctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	n, err := sqlcgen.New(cleanup).DeleteEndedSignInFailureBudgets(bctx, sqlcgen.DeleteEndedSignInFailureBudgetsParams{
		WindowEndedBefore: t0.Add(-15 * time.Minute), BatchSize: 1000,
	})
	cancel()
	if err != nil {
		t.Fatalf("cleanup batch while the /48 is locked: %v (it must skip the row, not wait)", err)
	}
	if n != 2 {
		t.Fatalf("cleanup batch deleted %d rows, want 2 (the /64 and the IPv4 budget; the locked /48 skipped)", n)
	}

	// (3) The sign-in continues to its /64, whose deletion is not committed yet: it waits for the batch.
	type result struct {
		row sqlcgen.ReserveSignInFailureRow
		err error
	}
	done := make(chan result, 1)
	go func() {
		row, err := reserve(ctx, signIn, subnet64, t0)
		done <- result{row, err}
	}()
	waitForLockWait(t, ctx, admin, signInPID)

	// (4) The batch commits; the sign-in proceeds (the /64 starts a new window) and commits.
	if err := cleanup.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	r := <-done
	if r.err != nil || r.row.FailureCount != 1 || !r.row.WindowStartAt.Equal(t0) {
		t.Fatalf("reserve /64 after the batch = %+v, %v; want a new window with 1 slot", r.row, r.err)
	}
	if err := signIn.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	for _, src := range []string{aggregate48, subnet64} {
		if n, start, ok := budgetRow(t, ctx, admin, src); !ok || n != 1 || !start.Equal(t0) {
			t.Errorf("%s = %d slots from %v (exists %v), want 1 slot from %v", src, n, start, ok, t0)
		}
	}
	if _, _, ok := budgetRow(t, ctx, admin, "198.51.100.1/32"); ok {
		t.Error("the ended IPv4 budget was not deleted")
	}
	// A full cleanup cycle now leaves the open windows alone.
	l := newLimiter(t, db, &clock{now: t0}, budgetsOnly(100, 100, 500))
	if res, err := l.Cleanup(ctx); err != nil || res.FailureBudgets != 0 || !res.Complete {
		t.Errorf("cleanup of open windows = %+v, %v; want nothing deleted", res, err)
	}
}

// A whole cleanup cycle (both tables, every batch) completes while sign-ins hold rows it would
// delete, without waiting for them; it deletes the rest, and the next cycle takes the skipped rows.
func TestIntegrationCleanupCycleDoesNotWaitForSignIns(t *testing.T) {
	t.Parallel()
	ctx, db, admin := setup(t)
	ended := t0.Add(-20 * time.Minute)
	for i := range 30 {
		seedBudget(t, ctx, admin, fmt.Sprintf("2001:db8:77:%x::/64", i+1), ended, 1)
	}
	seedBudget(t, ctx, admin, "2001:db8:77::/48", ended, 30)
	// The sign-in holds the /48 and two /64s (locked as a reservation would lock them).
	hold, err := db.Pool().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hold.Exec(ctx, `SELECT 1 FROM portal_sign_in_failure_budgets WHERE source IN ('2001:db8:77::/48', '2001:db8:77:1::/64', '2001:db8:77:2::/64') FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	l, err := New(db, budgetsOnly(100, 100, 500), WithClock(func() time.Time { return t0 }), WithCleanupBounds(4, 100))
	if err != nil {
		t.Fatal(err)
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	res, err := l.Cleanup(cctx)
	cancel()
	if err != nil || res.FailureBudgets != 28 || !res.Complete {
		t.Fatalf("cleanup while rows are held = %+v, %v; want the 28 unheld rows deleted without waiting", res, err)
	}
	if err := hold.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if res, err := l.Cleanup(ctx); err != nil || res.FailureBudgets != 3 {
		t.Errorf("next cleanup = %+v, %v; want the 3 skipped rows", res, err)
	}
}

// Control: the same interaction with the former unbatched DELETE deadlocks, so the test above
// reproduces the reviewed defect and would fail on the old cleanup.
func TestIntegrationFormerCleanupDeadlocked(t *testing.T) {
	t.Parallel()
	ctx, db, admin := setup(t)
	ended := t0.Add(-20 * time.Minute)
	seedBudget(t, ctx, admin, subnet64, ended, 7)
	seedBudget(t, ctx, admin, aggregate48, ended, 7)

	signIn, err := db.Pool().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = signIn.Rollback(ctx) }() //nolint:errcheck // cleanup
	if _, err := reserve(ctx, signIn, aggregate48, t0); err != nil {
		t.Fatal(err)
	}
	oldCleanup, err := db.Pool().Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer oldCleanup.Release()
	var oldPID int32
	if err := oldCleanup.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&oldPID); err != nil {
		t.Fatal(err)
	}
	deleted := make(chan error, 1)
	go func() {
		_, err := oldCleanup.Exec(ctx, `DELETE FROM portal_sign_in_failure_budgets WHERE window_start_at <= $1 OR failure_count = 0`, t0.Add(-15*time.Minute))
		deleted <- err
	}()
	waitForLockWait(t, ctx, admin, oldPID) // it deleted the /64 and now waits for the /48
	_, signInErr := reserve(ctx, signIn, subnet64, t0)
	cleanupErr := <-deleted
	if !isDeadlock(signInErr) && !isDeadlock(cleanupErr) {
		t.Fatalf("former cleanup: sign-in %v, cleanup %v; want one of them to fail with a deadlock (40P01)", signInErr, cleanupErr)
	}
}

func isDeadlock(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40P01"
}

// tally counts slots per source that the test expects to remain reserved.
type tally struct {
	mu sync.Mutex
	n  map[string]int
}

func (s *tally) add(src netip.Prefix, d int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n[src.String()] += d
}

// Concurrent IPv6 sign-ins (reservations, refunds and refusals) on many /64s of a few /48s, while
// three instances run cleanup cycles continuously with small batches; then the windows end and the
// same traffic runs again while the cleaners delete the ended rows. Every counter must equal the
// reservations made minus the refunds; nothing may fail (a deadlock would surface as an error).
func TestIntegrationCleanupUnderConcurrentIPv6SignIns(t *testing.T) {
	t.Parallel()
	ctx, roles, db, admin := setupRoles(t)
	clk := &clock{now: t0}
	const aggregateLimit = 150
	cfg := Config{FailuresPerIPv4: 1_000_000, FailuresPerIPv6Subnet: 1_000_000, FailuresPerIPv6Aggregate: aggregateLimit,
		FailureWindow: 15 * time.Minute, RequestsPerSecond: 1000, RequestBurst: 1_000_000}
	signIns := newLimiter(t, db, clk, cfg)

	// Ended budgets of the sources the traffic will use, each /64 stored before its /48 (the
	// deadlock-prone order), unrelated ended budgets and full buckets, and empty open budgets.
	const aggregates, subnets = 6, 24
	ended := t0.Add(-20 * time.Minute)
	for a := range aggregates {
		for s := range subnets {
			seedBudget(t, ctx, admin, fmt.Sprintf("2001:db8:%x:%x::/64", 0xa0+a, s+1), ended, 2)
		}
		seedBudget(t, ctx, admin, fmt.Sprintf("2001:db8:%x::/48", 0xa0+a), ended, 2*subnets)
	}
	for _, sql := range []string{
		`INSERT INTO portal_sign_in_failure_budgets (source, window_start_at, failure_count)
		 SELECT ('192.0.2.0'::inet + i)::cidr, $1, 1 FROM generate_series(1, 200) i`,
		`INSERT INTO portal_sign_in_request_buckets (source, tokens, refilled_at, last_allowed)
		 SELECT ('198.18.0.0'::inet + i)::cidr, 5, $1, true FROM generate_series(1, 2000) i`,
	} {
		if _, err := admin.Pool().Exec(ctx, sql, ended); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := admin.Pool().Exec(ctx, `INSERT INTO portal_sign_in_failure_budgets (source, window_start_at, failure_count)
		SELECT ('198.51.100.0'::inet + i)::cidr, $1, 0 FROM generate_series(1, 50) i`, t0.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	phase := func(name string, seed uint64) map[string]int {
		want := &tally{n: map[string]int{}}
		var refused, cycles atomic.Int64
		var errMu sync.Mutex
		var errs []error
		fail := func(err error) {
			errMu.Lock()
			defer errMu.Unlock()
			errs = append(errs, err)
		}
		stop := make(chan struct{})
		var cleaners sync.WaitGroup
		for i := range 3 { // three API instances, each with its own pool
			inst, err := New(open(t, ctx, roles.App), cfg, WithClock(clk.Now), WithCleanupBounds(16, 3))
			if err != nil {
				t.Fatal(err)
			}
			cleaners.Add(1)
			go func() {
				defer cleaners.Done()
				for {
					select {
					case <-stop:
						return
					default:
					}
					if _, err := inst.Cleanup(ctx); err != nil {
						fail(fmt.Errorf("%s: cleaner %d: %w", name, i, err))
						return
					}
					cycles.Add(1)
				}
			}()
		}
		var workers sync.WaitGroup
		for w := range 16 {
			workers.Add(1)
			go func() {
				defer workers.Done()
				rng := rand.New(rand.NewPCG(seed, uint64(w))) //nolint:gosec // deterministic test traffic
				for range 120 {
					ip := netip.MustParseAddr(fmt.Sprintf("2001:db8:%x:%x::%x", 0xa0+rng.IntN(aggregates), rng.IntN(subnets)+1, rng.IntN(1000)+1))
					a, err := signIns.Admit(ctx, ip)
					if err != nil {
						fail(fmt.Errorf("%s: admit: %w", name, err))
						return
					}
					if !a.Allowed {
						refused.Add(1)
						continue
					}
					for _, s := range a.Reservation.Slots {
						want.add(s.Source, 1)
					}
					if rng.IntN(3) == 0 { // a successful sign-in returns its slots
						if err := signIns.Refund(ctx, a.Reservation); err != nil {
							fail(fmt.Errorf("%s: refund: %w", name, err))
							return
						}
						for _, s := range a.Reservation.Slots {
							want.add(s.Source, -1)
						}
					}
				}
			}()
		}
		workers.Wait()
		close(stop)
		cleaners.Wait()
		for _, err := range errs {
			t.Errorf("%v (deadlock: %v)", err, isDeadlock(err))
		}
		t.Logf("%s: %d cleanup cycles ran during the traffic; %d attempts refused by the /48 limit", name, cycles.Load(), refused.Load())
		if cycles.Load() < 3 {
			t.Errorf("%s: only %d cleanup cycles overlapped the traffic", name, cycles.Load())
		}
		return want.n
	}

	check := func(name string, want map[string]int) {
		// Drain what the bounded cycles left, then compare every traffic source exactly.
		for range 100 {
			res, err := signIns.Cleanup(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if res.Complete {
				break
			}
		}
		rows, err := admin.Pool().Query(ctx, `SELECT source::text, failure_count, window_start_at FROM portal_sign_in_failure_budgets WHERE source << '2001:db8::/32'`)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]int{}
		for rows.Next() {
			var src string
			var n int
			var start time.Time
			if err := rows.Scan(&src, &n, &start); err != nil {
				t.Fatal(err)
			}
			if !start.Equal(clk.Now()) {
				t.Errorf("%s: %s has window %v, want the current %v (an ended window was not deleted, or an open one was reset)", name, src, start, clk.Now())
			}
			got[src] = n
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		for src, n := range want {
			if got[src] != n {
				t.Errorf("%s: %s holds %d slots, want %d (reserved minus refunded)", name, src, got[src], n)
			}
			if strings.HasSuffix(src, "/48") && n > aggregateLimit {
				t.Errorf("%s: %s holds %d slots, over the limit %d", name, src, n, aggregateLimit)
			}
		}
		for src := range got {
			if _, ok := want[src]; !ok {
				t.Errorf("%s: unexpected budget %s", name, src)
			}
		}
	}

	want := phase("phase 1", 1)
	check("phase 1", want)
	if n := count(t, ctx, admin, `SELECT count(*) FROM portal_sign_in_failure_budgets WHERE source << '192.0.2.0/24'`); n != 0 {
		t.Errorf("%d ended IPv4 budgets left, want 0", n)
	}
	if n := count(t, ctx, admin, `SELECT count(*) FROM portal_sign_in_request_buckets WHERE source << '198.18.0.0/15'`); n != 0 {
		t.Errorf("%d full buckets left, want 0", n)
	}
	if n := count(t, ctx, admin, `SELECT count(*) FROM portal_sign_in_failure_budgets WHERE source << '198.51.100.0/24' AND failure_count = 0`); n != 50 {
		t.Errorf("%d empty open budgets, want 50 (kept until their window ends)", n)
	}

	// Every window of phase 1 ends; the same traffic runs while the cleaners delete the ended rows.
	clk.Advance(16 * time.Minute)
	want = phase("phase 2", 2)
	check("phase 2", want)
}

// Several instances cleaning one large backlog at once share it: every row is deleted exactly once,
// and each cycle stops at its work bound.
func TestIntegrationConcurrentCleanersShareTheBacklog(t *testing.T) {
	t.Parallel()
	ctx, roles, db, admin := setupRoles(t)
	const rows = 6000
	for _, sql := range []string{
		`INSERT INTO portal_sign_in_failure_budgets (source, window_start_at, failure_count)
		 SELECT ('10.0.0.0'::inet + i)::cidr, $1, 1 FROM generate_series(1, $2::int) i`,
		`INSERT INTO portal_sign_in_request_buckets (source, tokens, refilled_at, last_allowed)
		 SELECT ('10.128.0.0'::inet + i)::cidr, 5, $1, true FROM generate_series(1, $2::int) i`,
	} {
		if _, err := admin.Pool().Exec(ctx, sql, t0.Add(-time.Hour), rows); err != nil {
			t.Fatal(err)
		}
	}
	// One cycle is bounded: 3 batches of 100 per table.
	bounded, err := New(db, budgetsOnly(100, 100, 500), WithClock(func() time.Time { return t0 }), WithCleanupBounds(100, 3))
	if err != nil {
		t.Fatal(err)
	}
	if res, err := bounded.Cleanup(ctx); err != nil || res.FailureBudgets != 300 || res.RequestBuckets != 300 || res.Complete {
		t.Fatalf("bounded cycle = %+v, %v; want 300 + 300 deleted, not complete", res, err)
	}
	var budgets, buckets atomic.Int64
	var wg sync.WaitGroup
	for range 4 {
		inst, err := New(open(t, ctx, roles.App), budgetsOnly(100, 100, 500), WithClock(func() time.Time { return t0 }), WithCleanupBounds(97, 1000))
		if err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := inst.Cleanup(ctx)
			if err != nil {
				t.Error(err)
				return
			}
			if !res.Complete {
				t.Error("an unbounded-enough cycle did not complete")
			}
			budgets.Add(res.FailureBudgets)
			buckets.Add(res.RequestBuckets)
		}()
	}
	wg.Wait()
	if budgets.Load() != rows-300 || buckets.Load() != rows-300 {
		t.Errorf("concurrent cleaners deleted %d budgets and %d buckets, want %d each (every row once)", budgets.Load(), buckets.Load(), rows-300)
	}
	if n := count(t, ctx, admin, `SELECT (SELECT count(*) FROM portal_sign_in_failure_budgets) + (SELECT count(*) FROM portal_sign_in_request_buckets)`); n != 0 {
		t.Errorf("%d rows left", n)
	}
}

// recorder is a CleanupMetrics that keeps the reported cycles.
type recorder struct {
	mu     sync.Mutex
	cycles []observed
}

type observed struct {
	succeeded, complete bool
	budgets, buckets    int64
}

func (r *recorder) ObserveSignInLimitCleanup(succeeded, complete bool, _ time.Duration, budgets, buckets int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cycles = append(r.cycles, observed{succeeded, complete, budgets, buckets})
}

// Every cycle is reported: a success with its deletions, a failure as a failure (never as a healthy
// cycle), and a cycle cut short by shutdown not at all.
func TestIntegrationCleanupCyclesAreReported(t *testing.T) {
	t.Parallel()
	ctx, db, admin := setup(t)
	seedBudget(t, ctx, admin, "203.0.113.50/32", t0.Add(-time.Hour), 1)
	rec := &recorder{}
	l, err := New(db, budgetsOnly(100, 100, 500), WithClock(func() time.Time { return t0 }), WithCleanupMetrics(rec))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.DiscardHandler)
	if !l.cleanupCycle(ctx, logger) {
		t.Fatal("cycle reported shutdown")
	}
	// The cleanup loses its DELETE privilege: the cycle fails and is reported as failed.
	if _, err := admin.Pool().Exec(ctx, `REVOKE DELETE ON portal_sign_in_failure_budgets FROM sms_app`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Pool().Exec(context.Background(), `GRANT DELETE ON portal_sign_in_failure_budgets TO sms_app`) //nolint:errcheck // test teardown
	})
	if !l.cleanupCycle(ctx, logger) {
		t.Fatal("failed cycle reported shutdown")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if l.cleanupCycle(cancelled, logger) {
		t.Error("a cycle after shutdown did not stop")
	}
	want := []observed{{true, true, 1, 0}, {false, false, 0, 0}}
	if fmt.Sprint(rec.cycles) != fmt.Sprint(want) {
		t.Errorf("reported cycles %+v, want %+v", rec.cycles, want)
	}
}

// cleanupQuery returns the SQL of a named query in queries/signinlimit.sql with its named parameters
// replaced by positional ones (in the order given), so the test plans exactly the committed text.
func cleanupQuery(t *testing.T, name string, params ...string) string {
	t.Helper()
	b, err := os.ReadFile("../../queries/signinlimit.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(b), "-- name: "+name+" ")
	if !ok {
		t.Fatalf("query %s not found", name)
	}
	if i := strings.Index(rest, "-- name: "); i >= 0 {
		rest = rest[:i]
	}
	var sql []string
	for _, line := range strings.Split(rest, "\n")[1:] {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			sql = append(sql, line)
		}
	}
	q := strings.Join(sql, "\n")
	for i, p := range params {
		q = strings.ReplaceAll(q, "@"+p, fmt.Sprintf("$%d", i+1))
	}
	if strings.Contains(q, "@") {
		t.Fatalf("query %s has unreplaced parameters: %s", name, q)
	}
	return q
}

// The cleanup batches use their indexes on a large table: the batch is chosen through the time
// index and deleted through the primary key, never by a scan of the whole table.
func TestIntegrationCleanupQueryPlansUseIndexes(t *testing.T) {
	t.Parallel()
	ctx, _, admin := setup(t)
	for _, sql := range []string{
		`INSERT INTO portal_sign_in_failure_budgets (source, window_start_at, failure_count)
		 SELECT ('10.0.0.0'::inet + i)::cidr, $1::timestamptz - (i % 1800) * interval '1 second', 1 + i % 9 FROM generate_series(1, 50000) i`,
		`INSERT INTO portal_sign_in_request_buckets (source, tokens, refilled_at, last_allowed)
		 SELECT ('10.0.0.0'::inet + i)::cidr, 5, $1::timestamptz - (i % 60) * interval '1 second', true FROM generate_series(1, 50000) i`,
		`ANALYZE portal_sign_in_failure_budgets, portal_sign_in_request_buckets`,
	} {
		var err error
		if strings.Contains(sql, "$1") {
			_, err = admin.Pool().Exec(ctx, sql, t0)
		} else {
			_, err = admin.Pool().Exec(ctx, sql)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	plan := func(sql string, args ...any) string {
		t.Helper()
		tx, err := admin.Pool().Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // plans only
		rows, err := tx.Query(ctx, "EXPLAIN (COSTS OFF) "+sql, args...)
		if err != nil {
			t.Fatal(err)
		}
		lines, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(lines, "\n")
	}
	for _, tt := range []struct {
		name, sql string
		args      []any
		indexes   []string
	}{
		{"budgets", cleanupQuery(t, "DeleteEndedSignInFailureBudgets", "window_ended_before", "batch_size"),
			[]any{t0.Add(-15 * time.Minute), int32(1000)},
			[]string{"ix_portal_sign_in_failure_budgets__window_start_at", "pk_portal_sign_in_failure_budgets"}},
		{"buckets", cleanupQuery(t, "DeleteFullSignInRequestBuckets", "full_before", "report_interval_before", "batch_size"),
			[]any{t0.Add(-10 * time.Second), t0.Add(-ReportInterval), int32(1000)},
			[]string{"ix_portal_sign_in_request_buckets__refilled_at", "pk_portal_sign_in_request_buckets"}},
	} {
		p := plan(tt.sql, tt.args...)
		for _, ix := range tt.indexes {
			if !strings.Contains(p, ix) {
				t.Errorf("%s: plan does not use %s:\n%s", tt.name, ix, p)
			}
		}
		if strings.Contains(p, "Seq Scan") || strings.Contains(p, "Bitmap Heap Scan") {
			t.Errorf("%s: plan scans the table:\n%s", tt.name, p)
		}
		if !strings.Contains(p, "LockRows") {
			t.Errorf("%s: plan does not lock its batch:\n%s", tt.name, p)
		}
	}
}
