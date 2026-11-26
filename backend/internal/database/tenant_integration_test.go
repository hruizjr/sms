//go:build integration

package database_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/database/rlsguard"
)

// The tenant-context helper InTenantTx (ADR-0010, RLS-2; decision log §18). The tests connect as
// the application role through a pool of ONE connection, so every transaction reuses the same
// server session and any leaked setting would be visible to the next one.

// tenantFixture is a migrated database with a test-only tenant-owned table protected as ADR-0010
// requires, owned by the migration role like a real table, and two tenant IDs from uuidv7().
type tenantFixture struct {
	roles   dbtest.Roles
	admin   *database.DB
	app     *database.DB // sms_app, one connection, RequireApplicationRole
	tenantA [16]byte
	tenantB [16]byte
}

func newTenantFixture(t *testing.T) tenantFixture {
	t.Helper()
	ctx := testContext(t)
	roles := dbtest.NewMigratedDatabase(t)
	f := tenantFixture{roles: roles, admin: open(t, roles.Admin)}
	mustExec(t, ctx, f.admin,
		`CREATE TABLE public.rls_test_notes (
		     tenant_id uuid NOT NULL,
		     note      text NOT NULL
		 )`,
		`ALTER TABLE public.rls_test_notes OWNER TO sms_migrator`,
		`ALTER TABLE public.rls_test_notes ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.rls_test_notes FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY tenant_isolation ON public.rls_test_notes
		     USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
		     WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)`,
		`GRANT SELECT, INSERT ON public.rls_test_notes TO sms_app`)

	cfg := roles.App
	cfg.MaxConns, cfg.MinConns = 1, 0
	app, err := database.Open(ctx, cfg, "sms-tenant-context-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	f.app = app
	f.tenantA = value[[16]byte](t, ctx, f.admin, `SELECT uuidv7()`)
	f.tenantB = value[[16]byte](t, ctx, f.admin, `SELECT uuidv7()`)
	return f
}

// setting returns app.tenant_id as seen by q ("" when unset or cleared).
func setting(t *testing.T, ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) string {
	t.Helper()
	var v *string
	if err := q.QueryRow(ctx, `SELECT current_setting('app.tenant_id', true)`).Scan(&v); err != nil {
		t.Fatalf("read app.tenant_id: %v", err)
	}
	if v == nil {
		return ""
	}
	return *v
}

func notes(t *testing.T, ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}) []string {
	t.Helper()
	rows, err := q.Query(ctx, `SELECT note FROM rls_test_notes ORDER BY note`)
	if err != nil {
		t.Fatalf("select notes: %v", err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("select notes: %v", err)
	}
	return got
}

func uuidText(id [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
}

func TestIntegrationInTenantTxRejectsInvalidIDsBeforeSQL(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	f := newTenantFixture(t)
	before := f.app.Pool().Stat().AcquireCount()
	badVersion := f.tenantA
	badVersion[6] &= 0x0f // version 0
	for _, id := range [][16]byte{{}, badVersion} {
		called := false
		err := f.app.InTenantTx(ctx, id, func(pgx.Tx) error { called = true; return nil })
		if !errors.Is(err, database.ErrInvalidTenantID) || called {
			t.Errorf("InTenantTx(%x) = %v, fn called = %v; want ErrInvalidTenantID and no call", id, err, called)
		}
	}
	if after := f.app.Pool().Stat().AcquireCount(); after != before {
		t.Errorf("invalid tenant IDs acquired %d connections, want 0", after-before)
	}
}

// TestIntegrationInTenantTxSettingIsTransactionLocal: the setting is visible inside the transaction
// and gone afterwards on the same connection, whether the transaction commits, fn returns an error,
// a statement fails, or fn panics.
func TestIntegrationInTenantTxSettingIsTransactionLocal(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	f := newTenantFixture(t)
	errFromFn := errors.New("fn failed")

	cases := []struct {
		name    string
		fn      func(tx pgx.Tx) error
		wantErr func(error) bool
	}{
		{"commit", func(pgx.Tx) error { return nil }, func(err error) bool { return err == nil }},
		{"fn returns an error (rollback)", func(pgx.Tx) error { return errFromFn },
			func(err error) bool { return errors.Is(err, errFromFn) }},
		{"statement fails (rollback)", func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `SELECT 1/0`)
			return err
		}, func(err error) bool {
			var pgErr *pgconn.PgError
			return errors.As(err, &pgErr) && pgErr.Code == "22012"
		}},
	}
	for _, tc := range cases {
		var inside string
		err := f.app.InTenantTx(ctx, f.tenantA, func(tx pgx.Tx) error {
			inside = setting(t, ctx, tx)
			return tc.fn(tx)
		})
		if !tc.wantErr(err) {
			t.Errorf("%s: InTenantTx() = %v", tc.name, err)
		}
		if inside != uuidText(f.tenantA) {
			t.Errorf("%s: setting inside = %q, want %q", tc.name, inside, uuidText(f.tenantA))
		}
		if after := setting(t, ctx, f.app.Pool()); after != "" {
			t.Errorf("%s: setting after the transaction = %q on the reused connection, want empty", tc.name, after)
		}
	}

	// A panic in fn rolls back, is re-raised, and leaves no setting behind.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic in fn was not re-raised")
			}
		}()
		_ = f.app.InTenantTx(ctx, f.tenantA, func(pgx.Tx) error { panic("boom") })
	}()
	if after := setting(t, ctx, f.app.Pool()); after != "" {
		t.Errorf("after a panic: setting = %q, want empty", after)
	}

	// Never a session-level value: pg_settings shows no value once the transaction has ended.
	if src := value[string](t, ctx, f.app, `SELECT coalesce((SELECT source FROM pg_settings WHERE name = 'app.tenant_id'), 'none')`); src == "session" {
		t.Errorf("app.tenant_id has a session-level source")
	}
}

func TestIntegrationInTenantTxEnforcesRowLevelSecurity(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	f := newTenantFixture(t)

	insert := func(tenant [16]byte, rowTenant [16]byte, note string) error {
		return f.app.InTenantTx(ctx, tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO rls_test_notes (tenant_id, note) VALUES ($1, $2)`, rowTenant, note)
			return err
		})
	}
	for _, n := range []struct {
		tenant [16]byte
		note   string
	}{{f.tenantA, "a1"}, {f.tenantA, "a2"}, {f.tenantB, "b1"}} {
		if err := insert(n.tenant, n.tenant, n.note); err != nil {
			t.Fatalf("insert %s: %v", n.note, err)
		}
	}

	// Each tenant sees only its own rows (the same connection serves both).
	for _, tc := range []struct {
		tenant [16]byte
		want   []string
	}{{f.tenantA, []string{"a1", "a2"}}, {f.tenantB, []string{"b1"}}, {f.tenantA, []string{"a1", "a2"}}} {
		var got []string
		if err := f.app.InTenantTx(ctx, tc.tenant, func(tx pgx.Tx) error { got = notes(t, ctx, tx); return nil }); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("tenant %s sees %v, want %v", uuidText(tc.tenant), got, tc.want)
		}
	}

	// Writing another tenant's row is rejected by the policy's WITH CHECK.
	err := insert(f.tenantA, f.tenantB, "forged")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Errorf("insert for tenant B under tenant A's context: %v, want a row-level security violation", err)
	}

	// Without tenant context the application role sees nothing (fail closed).
	if got := notes(t, ctx, f.app.Pool()); len(got) != 0 {
		t.Errorf("without tenant context: %v, want no rows", got)
	}

	// Queries must use the transaction given to fn: a query through another connection inside fn
	// has no tenant context and sees nothing. (The test pool has one connection, so a second pool
	// is used; the shared pool would otherwise wait for the connection fn holds.)
	other := open(t, f.roles.App)
	if err := f.app.InTenantTx(ctx, f.tenantA, func(pgx.Tx) error {
		if got := notes(t, ctx, other.Pool()); len(got) != 0 {
			t.Errorf("query outside the transaction saw %v, want no rows", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// The guard accepts this table as protected.
	violations, err := rlsguard.Check(ctx, f.admin.Pool())
	if err != nil || len(violations) != 0 {
		t.Errorf("rlsguard.Check() = %v, %v; want no violations", violations, err)
	}
}

// TestIntegrationInTenantTxNoLeakUnderConcurrency: many goroutines alternate tenants over a small
// pool; inside every transaction the setting and the visible rows belong to that transaction's
// tenant only.
func TestIntegrationInTenantTxNoLeakUnderConcurrency(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	f := newTenantFixture(t)
	for _, n := range []struct {
		tenant [16]byte
		note   string
	}{{f.tenantA, "a"}, {f.tenantB, "b"}} {
		if err := f.app.InTenantTx(ctx, n.tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO rls_test_notes (tenant_id, note) VALUES ($1, $2)`, n.tenant, n.note)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}

	cfg := f.roles.App
	cfg.MaxConns = 3
	pool, err := database.Open(ctx, cfg, "sms-tenant-concurrency-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var wg sync.WaitGroup
	for w := range 8 {
		wg.Go(func() {
			for i := range 25 {
				tenant, want := f.tenantA, "a"
				if (w+i)%2 == 1 {
					tenant, want = f.tenantB, "b"
				}
				err := pool.InTenantTx(ctx, tenant, func(tx pgx.Tx) error {
					if got := setting(t, ctx, tx); got != uuidText(tenant) {
						return fmt.Errorf("setting %q, want %q", got, uuidText(tenant))
					}
					if got := notes(t, ctx, tx); len(got) != 1 || got[0] != want {
						return fmt.Errorf("visible rows %v, want [%s]", got, want)
					}
					return nil
				})
				if err != nil {
					t.Errorf("worker %d, iteration %d: %v", w, i, err)
					return
				}
			}
		})
	}
	wg.Wait()
	// Every pooled connection is clean afterwards: hold all of them at once and look at each.
	var conns []*pgxpool.Conn
	for range cfg.MaxConns {
		conn, err := pool.Pool().Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, conn)
	}
	for _, conn := range conns {
		if got := setting(t, ctx, conn); got != "" {
			t.Errorf("pooled connection still has app.tenant_id = %q", got)
		}
		conn.Release()
	}
}
