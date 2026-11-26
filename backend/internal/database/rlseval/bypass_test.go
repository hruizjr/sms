//go:build integration && rlseval

package rlseval

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ADR-0010 evaluation requirement 6 ("the application role cannot bypass RLS"), and the conditions
// under which row-level security does not apply at all.

// noForceTable creates rlseval.items_noforce: the same as rlseval.items, with RLS enabled but NOT
// forced, so that its owner bypasses it.
func (f *fixture) noForceTable(t testing.TB) {
	t.Helper()
	f.ownerExec(t,
		`CREATE TABLE rlseval.items_noforce (tenant_id uuid NOT NULL, label text NOT NULL)`,
		`ALTER TABLE rlseval.items_noforce ENABLE ROW LEVEL SECURITY`,
		`CREATE POLICY tenant_isolation ON rlseval.items_noforce TO `+ident(f.app)+
			` USING (`+tenantPolicy+`) WITH CHECK (`+tenantPolicy+`)`,
		`GRANT SELECT ON rlseval.items_noforce TO `+ident(f.app),
	)
	f.adminExec(t, `INSERT INTO rlseval.items_noforce SELECT tenant_id, label FROM rlseval.items`)
}

// TestWhoBypassesRLS: which roles see which rows of a table with RLS enabled, with and without
// FORCE ROW LEVEL SECURITY, when no tenant context is set (and when it is, for the application).
func TestWhoBypassesRLS(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	f.noForceTable(t)
	all := rowsOf(tenantA, tenantB)

	cases := []struct {
		name   string
		pool   *pgxpool.Pool
		tenant string
		table  string
		want   []string
	}{
		{"superuser, forced table: bypasses", f.admin, "", "rlseval.items", all},
		{"superuser, unforced table: bypasses", f.admin, "", "rlseval.items_noforce", all},
		{"owner, unforced table: bypasses", f.pool(t, f.owner, 1, nil), "", "rlseval.items_noforce", all},
		{"owner, forced table, no owner policy: default deny", f.pool(t, f.owner, 1, nil), "", "rlseval.items", nil},
		{"BYPASSRLS role: bypasses", f.pool(t, f.bypass, 1, nil), "", "rlseval.items", all},
		{"app, no context: no rows", f.pool(t, f.app, 1, nil), "", "rlseval.items", nil},
		{"app, tenant A: tenant A only", f.pool(t, f.app, 1, nil), tenantA, "rlseval.items", rowsOf(tenantA)},
		{"app, unforced table, tenant B: tenant B only", f.pool(t, f.app, 1, nil), tenantB, "rlseval.items_noforce", rowsOf(tenantB)},
	}
	for _, c := range cases {
		err := inTenant(ctx, c.pool, c.tenant, func(tx pgx.Tx) error {
			got, err := visible(ctx, tx, c.table)
			if err != nil {
				return err
			}
			if !equalRows(got, c.want) {
				t.Errorf("%s: visible = %v, want %v", c.name, got, c.want)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
}

// TestApplicationRoleCannotBypass: the application role is not superuser, has no BYPASSRLS, does
// not own the table, cannot switch row security off, and cannot disable RLS, drop the policy, or
// give itself BYPASSRLS.
func TestApplicationRoleCannotBypass(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	app := f.pool(t, f.app, 1, nil)

	var super, bypass, owns bool
	err := app.QueryRow(ctx, `
		SELECT r.rolsuper, r.rolbypassrls, c.relowner = r.oid
		FROM pg_roles r, pg_class c
		WHERE r.rolname = current_user AND c.oid = 'rlseval.items'::regclass`).Scan(&super, &bypass, &owns)
	if err != nil {
		t.Fatal(err)
	}
	if super || bypass || owns {
		t.Fatalf("application role: superuser=%v bypassrls=%v owner=%v, want all false", super, bypass, owns)
	}

	// row_security = off does not bypass anything for a role subject to RLS: the query fails.
	err = pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL row_security = off`); err != nil {
			return err
		}
		_, err := visible(ctx, tx, "rlseval.items")
		return err
	})
	wantSQLState(t, err, sqlStateInsufficientPrivilege) // query would be affected by row-level security policy

	for _, stmt := range []string{
		`ALTER TABLE rlseval.items DISABLE ROW LEVEL SECURITY`,
		`ALTER TABLE rlseval.items NO FORCE ROW LEVEL SECURITY`,
		`DROP POLICY items_tenant_isolation ON rlseval.items`,
		`CREATE POLICY open ON rlseval.items USING (true)`,
		`ALTER ROLE ` + ident(f.app) + ` BYPASSRLS`,
		`ALTER TABLE rlseval.items OWNER TO ` + ident(f.app),
	} {
		_, err := app.Exec(ctx, stmt)
		wantSQLState(t, err, sqlStateInsufficientPrivilege)
	}

	// The superuser with row_security = off still sees everything: the setting only turns bypass
	// into an error for roles that would otherwise be filtered.
	err = pgx.BeginFunc(ctx, f.admin, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL row_security = off`); err != nil {
			return err
		}
		got, err := visible(ctx, tx, "rlseval.items")
		if err == nil && len(got) != 5 {
			t.Errorf("superuser with row_security = off saw %d rows, want 5", len(got))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestTruncateIgnoresRLS: TRUNCATE is not subject to row-level security. A role with the TRUNCATE
// privilege empties the table for every tenant, whatever its tenant context. The application role
// must never be granted TRUNCATE.
func TestTruncateIgnoresRLS(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	app := f.pool(t, f.app, 1, nil)

	err := inTenant(ctx, app, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `TRUNCATE rlseval.items`)
		return err
	})
	wantSQLState(t, err, sqlStateInsufficientPrivilege) // not granted: refused

	f.ownerExec(t, `GRANT TRUNCATE ON rlseval.items TO `+ident(f.app))
	err = inTenant(ctx, app, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `TRUNCATE rlseval.items`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := visible(ctx, f.admin, "rlseval.items")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("after TRUNCATE in tenant A's context %d rows remain; expected the demonstrated loss of all tenants' rows", len(got))
	}
}

// TestContextIsChosenByTrustedCode records the threat-model limit (rls-evaluation.md, §3): the
// tenant setting is an ordinary custom setting. Any SQL running as the application role, including
// SQL from a compromised application or a stolen application credential, can set it to any tenant
// and then read that tenant's rows. RLS here protects against missing or wrong tenant filters in
// trusted code, not against an attacker who can run SQL as the application role.
func TestContextIsChosenByTrustedCode(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	app := f.pool(t, f.app, 1, nil)

	err := inTenant(ctx, app, tenantA, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenantB); err != nil {
			return err
		}
		got, err := visible(ctx, tx, "rlseval.items")
		if err != nil {
			return err
		}
		if !equalRows(got, rowsOf(tenantB)) {
			t.Errorf("after switching the setting to tenant B: %v; expected the demonstrated access to tenant B", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
