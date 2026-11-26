//go:build integration && rlseval

package rlseval

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A TEST-ONLY sketch of the transaction helper that product code would need if RLS were adopted.
// It is not product code, is not exported, and must not be copied into internal/database without
// the owner's adoption decision (ADR-0010) and a separate, approved task.
//
// Shape: like database.DB.InTx, but the first statement of every transaction sets the tenant
// transaction-locally. sqlc's generated New(db DBTX) accepts a pgx.Tx, so generated queries would
// run inside fn unchanged.

var errInvalidTenant = errors.New("tenant ID is not a valid UUID")

func withTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	var id pgtype.UUID
	if err := id.Scan(tenantID); err != nil || !id.Valid {
		return errInvalidTenant // validated in Go, before any SQL runs
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		// set_config with a bound parameter: SET LOCAL cannot take parameters, and building the value
		// into the SQL text would be injection-prone. ::uuid::text stores the canonical form.
		if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1::uuid::text, true)`, id); err != nil {
			return fmt.Errorf("set tenant context: %w", err)
		}
		return fn(tx)
	})
}

func TestWithTenantSketch(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	app := f.pool(t, f.app, 1, nil)

	for _, bad := range []string{"", "not-a-uuid", "' OR true --", tenantA + "x"} {
		called := false
		err := withTenant(ctx, app, bad, func(pgx.Tx) error { called = true; return nil })
		if !errors.Is(err, errInvalidTenant) || called {
			t.Errorf("withTenant(%q): err = %v, fn called = %v; want errInvalidTenant before any SQL", bad, err, called)
		}
	}

	// Upper-case input is stored in canonical form, and the policy matches it.
	err := withTenant(ctx, app, "BBBBBBBB-0000-4000-8000-00000000000B", func(tx pgx.Tx) error {
		v, _, err := currentTenantSetting(ctx, tx)
		if err != nil {
			return err
		}
		if v != tenantB {
			t.Errorf("setting = %q, want canonical %q", v, tenantB)
		}
		got, err := visible(ctx, tx, "rlseval.items")
		if err != nil {
			return err
		}
		if !equalRows(got, rowsOf(tenantB)) {
			t.Errorf("visible = %v, want tenant B's rows", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	assertNoContext(ctx, t, app)
}
