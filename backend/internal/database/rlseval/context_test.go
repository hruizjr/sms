//go:build integration && rlseval

package rlseval

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ADR-0010 evaluation requirement 1: transaction-scoped tenant context.

// currentTenantSetting returns current_setting('app.tenant_id', true) and whether it was NULL.
func currentTenantSetting(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (string, bool, error) {
	var v *string
	if err := q.QueryRow(ctx, `SELECT current_setting('app.tenant_id', true)`).Scan(&v); err != nil {
		return "", false, err
	}
	if v == nil {
		return "", true, nil
	}
	return *v, false, nil
}

func backendPID(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (int32, error) {
	var pid int32
	err := q.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid)
	return pid, err
}

// TestContextIsTransactionLocal: set_config(..., true) applies inside the transaction only, on
// commit and on rollback, and the next transaction on the same connection has no tenant context.
//
// It also records a PostgreSQL behaviour the policy must handle: before the setting is ever used
// on a connection, current_setting('app.tenant_id', true) returns NULL; after a transaction-local
// value has ended, it returns the empty string (""). A policy that casts the value without NULLIF
// therefore fails with an error instead of matching nothing (failclosed_test.go).
func TestContextIsTransactionLocal(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()

	if _, isNull, err := currentTenantSetting(ctx, conn); err != nil || !isNull {
		t.Fatalf("fresh connection: setting NULL = %v (err %v), want NULL", isNull, err)
	}

	for _, end := range []string{"COMMIT", "ROLLBACK"} {
		if _, err := conn.Exec(ctx, "BEGIN"); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenantA); err != nil {
			t.Fatal(err)
		}
		got, err := visible(ctx, conn, "rlseval.items")
		if err != nil {
			t.Fatal(err)
		}
		if !equalRows(got, rowsOf(tenantA)) {
			t.Fatalf("inside transaction: visible = %v, want tenant A's rows", got)
		}
		if _, err := conn.Exec(ctx, end); err != nil {
			t.Fatal(err)
		}

		v, isNull, err := currentTenantSetting(ctx, conn)
		if err != nil {
			t.Fatal(err)
		}
		if isNull || v != "" {
			t.Fatalf("after %s: setting = %q (NULL %v), want the empty string", end, v, isNull)
		}
		got, err = visible(ctx, conn, "rlseval.items")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("after %s: visible without context = %v, want none", end, got)
		}
	}
}

// TestContextRolledBackWithSavepoint: a tenant set inside a savepoint that is rolled back reverts
// to the value set before the savepoint.
func TestContextRolledBackWithSavepoint(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	err := inTenant(ctx, pool, tenantA, func(tx pgx.Tx) error {
		sp, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := sp.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenantB); err != nil {
			return err
		}
		if err := sp.Rollback(ctx); err != nil {
			return err
		}
		v, _, err := currentTenantSetting(ctx, tx)
		if err != nil {
			return err
		}
		if v != tenantA {
			t.Errorf("after rolling back the savepoint: setting = %q, want tenant A", v)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestSessionLevelContextLeaks is the negative case that explains why session-level settings are
// forbidden: set_config(..., false) (or SET app.tenant_id) outlives the transaction, so the next
// transaction on the same pooled connection, for any request, runs as the previous tenant.
func TestSessionLevelContextLeaks(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	for _, set := range []string{
		`SELECT set_config('app.tenant_id', '` + tenantA + `', false)`,
		`SET app.tenant_id = '` + tenantA + `'`,
	} {
		// "Request 1" sets the tenant for the session and returns the connection to the pool.
		if _, err := pool.Exec(ctx, set); err != nil {
			t.Fatal(err)
		}
		// "Request 2" does not set any tenant, and still sees tenant A's rows: a leak.
		var got []string
		err := inTenant(ctx, pool, "", func(tx pgx.Tx) error {
			var err error
			got, err = visible(ctx, tx, "rlseval.items")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if !equalRows(got, rowsOf(tenantA)) {
			t.Fatalf("%s: the next transaction saw %v; expected the demonstrated leak of tenant A's rows", set, got)
		}
		if _, err := pool.Exec(ctx, `RESET app.tenant_id`); err != nil {
			t.Fatal(err)
		}
	}
}

// TestContextClearedAfterFailedTransaction: an error inside the transaction rolls it back, and a
// cancelled context aborts it; in both cases no later transaction inherits the tenant.
func TestContextClearedAfterFailedTransaction(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	errFromRequest := errors.New("request failed")
	err := inTenant(ctx, pool, tenantA, func(pgx.Tx) error { return errFromRequest })
	if !errors.Is(err, errFromRequest) {
		t.Fatalf("err = %v, want the request error", err)
	}

	cancelled, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	err = inTenant(cancelled, pool, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(cancelled, `SELECT pg_sleep(5)`)
		return err
	})
	if err == nil {
		t.Fatal("the cancelled transaction succeeded")
	}

	assertNoContext(ctx, t, pool)
}

func assertNoContext(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	err := inTenant(ctx, pool, "", func(tx pgx.Tx) error {
		got, err := visible(ctx, tx, "rlseval.items")
		if err != nil {
			return err
		}
		if len(got) != 0 {
			t.Errorf("a later transaction without context saw %v, want none", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
