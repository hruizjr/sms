//go:build integration && rlseval

package rlseval

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// ADR-0010 evaluation requirement 3: fail closed when the tenant context is missing or wrong.

// TestMissingContextFailsClosed: without a tenant context the application role sees no rows,
// cannot insert, and its UPDATE and DELETE statements affect no rows. UPDATE and DELETE fail
// silently (no error, zero rows affected), so callers must check the affected-row count.
func TestMissingContextFailsClosed(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	// Twice: first on a fresh connection (setting is NULL), then after the setting has been used
	// once on that connection (setting is '').
	for round := range 2 {
		err := inTenant(ctx, pool, "", func(tx pgx.Tx) error {
			got, err := visible(ctx, tx, "rlseval.items")
			if err != nil {
				return err
			}
			if len(got) != 0 {
				t.Errorf("round %d: SELECT without context saw %v", round, got)
			}
			for _, stmt := range []string{
				`UPDATE rlseval.items SET label = 'changed'`,
				`DELETE FROM rlseval.items`,
			} {
				tag, err := tx.Exec(ctx, stmt)
				if err != nil {
					return err
				}
				if tag.RowsAffected() != 0 {
					t.Errorf("round %d: %s affected %d rows without context", round, stmt, tag.RowsAffected())
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		err = inTenant(ctx, pool, "", func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO rlseval.items (tenant_id, label) VALUES ($1, 'x')`, tenantA)
			return err
		})
		wantSQLState(t, err, sqlStateInsufficientPrivilege) // new row violates row-level security policy

		// Use the setting once so that round 2 starts with '' rather than NULL.
		if err := inTenant(ctx, pool, tenantC, func(pgx.Tx) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	got, err := visible(ctx, f.admin, "rlseval.items")
	if err != nil {
		t.Fatal(err)
	}
	if !equalRows(got, rowsOf(tenantA, tenantB)) {
		t.Fatalf("seeded rows changed: %v", got)
	}
}

// TestWithCheckBlocksCrossTenantWrites: in tenant A's context the application role cannot insert a
// row for tenant B, cannot move its own row to tenant B, and cannot touch tenant B's rows.
func TestWithCheckBlocksCrossTenantWrites(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	err := inTenant(ctx, pool, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO rlseval.items (tenant_id, label) VALUES ($1, 'planted')`, tenantB)
		return err
	})
	wantSQLState(t, err, sqlStateInsufficientPrivilege)

	err = inTenant(ctx, pool, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE rlseval.items SET tenant_id = $1 WHERE label = 'a1'`, tenantB)
		return err
	})
	wantSQLState(t, err, sqlStateInsufficientPrivilege)

	err = inTenant(ctx, pool, tenantA, func(tx pgx.Tx) error {
		for _, stmt := range []string{
			`UPDATE rlseval.items SET label = 'changed' WHERE label = 'b1'`,
			`DELETE FROM rlseval.items WHERE label = 'b2'`,
		} {
			tag, err := tx.Exec(ctx, stmt)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 0 {
				t.Errorf("tenant A: %s affected %d of tenant B's rows", stmt, tag.RowsAffected())
			}
		}
		// Its own rows remain writable.
		tag, err := tx.Exec(ctx, `UPDATE rlseval.items SET label = label WHERE tenant_id = $1`, tenantA)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 3 {
			t.Errorf("tenant A updated %d of its own rows, want 3", tag.RowsAffected())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestMalformedContextIsAnError: a tenant value that is not a UUID makes every statement on the
// table fail (invalid input syntax for type uuid) rather than match anything.
func TestMalformedContextIsAnError(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	pool := f.pool(t, f.app, 1, nil)

	for _, bad := range []string{"not-a-uuid", "' OR true --", tenantA + "," + tenantB} {
		err := inTenant(ctx, pool, bad, func(tx pgx.Tx) error {
			_, err := visible(ctx, tx, "rlseval.items")
			return err
		})
		wantSQLState(t, err, sqlStateInvalidText)
	}
}

// TestPolicyExpressionVariants compares three ways to read the setting in a policy:
//
//	raw     current_setting(...)::uuid      NULL on a fresh connection -> no rows;
//	                                        '' after first use        -> ERROR (inconsistent)
//	nullif  NULLIF(current_setting(...), '')::uuid   no rows in both cases (consistent, silent)
//	require rlseval.require_tenant_id()     ERROR in both cases (consistent, loud)
func TestPolicyExpressionVariants(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)

	f.ownerExec(t,
		`CREATE FUNCTION rlseval.require_tenant_id() RETURNS uuid
			LANGUAGE plpgsql STABLE PARALLEL SAFE AS $$
		DECLARE v text := current_setting('app.tenant_id', true);
		BEGIN
			IF v IS NULL OR v = '' THEN
				RAISE EXCEPTION 'tenant context is not set' USING ERRCODE = 'P0001';
			END IF;
			RETURN v::uuid;
		END $$`,
		`GRANT EXECUTE ON FUNCTION rlseval.require_tenant_id() TO `+ident(f.app),
	)
	variants := map[string]string{
		"raw":     `tenant_id = current_setting('app.tenant_id', true)::uuid`,
		"nullif":  tenantPolicy,
		"require": `tenant_id = rlseval.require_tenant_id()`,
	}
	for name, expr := range variants {
		f.ownerExec(t,
			`CREATE TABLE rlseval.items_`+name+` (tenant_id uuid NOT NULL, label text NOT NULL)`,
			`ALTER TABLE rlseval.items_`+name+` ENABLE ROW LEVEL SECURITY`,
			`ALTER TABLE rlseval.items_`+name+` FORCE ROW LEVEL SECURITY`,
			`CREATE POLICY tenant_isolation ON rlseval.items_`+name+` TO `+ident(f.app)+` USING (`+expr+`)`,
			`GRANT SELECT ON rlseval.items_`+name+` TO `+ident(f.app),
		)
		f.adminExec(t, `INSERT INTO rlseval.items_`+name+` SELECT tenant_id, label FROM rlseval.items`)
	}

	type outcome struct {
		rows    int
		sqlCode string
	}
	query := func(table string) outcome {
		pool := f.pool(t, f.app, 1, nil) // a fresh pool: its first connection has never used the setting
		var out [2]outcome
		for round := range 2 {
			err := inTenant(ctx, pool, "", func(tx pgx.Tx) error {
				got, err := visible(ctx, tx, table)
				out[round].rows = len(got)
				return err
			})
			if err != nil {
				out[round].sqlCode = wantAnySQLState(t, err)
			}
			if err := inTenant(ctx, pool, tenantC, func(pgx.Tx) error { return nil }); err != nil {
				t.Fatal(err)
			}
		}
		if out[0] != out[1] {
			return outcome{rows: -1, sqlCode: "fresh=" + out[0].sqlCode + "/used=" + out[1].sqlCode}
		}
		return out[0]
	}

	want := map[string]outcome{
		"raw":     {rows: -1, sqlCode: "fresh=/used=" + sqlStateInvalidText},
		"nullif":  {rows: 0},
		"require": {sqlCode: sqlStateMissingTenant},
	}
	for name := range variants {
		got := query("rlseval.items_" + name)
		if got != want[name] {
			t.Errorf("variant %s without context: got %+v, want %+v", name, got, want[name])
		}
		// With a context, all three variants return exactly the tenant's rows.
		pool := f.pool(t, f.app, 1, nil)
		err := inTenant(ctx, pool, tenantB, func(tx pgx.Tx) error {
			rows, err := visible(ctx, tx, "rlseval.items_"+name)
			if err != nil {
				return err
			}
			if !equalRows(rows, rowsOf(tenantB)) {
				t.Errorf("variant %s with tenant B: %v", name, rows)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func wantAnySQLState(t testing.TB, err error) string {
	t.Helper()
	return asPgError(t, err).Code
}
