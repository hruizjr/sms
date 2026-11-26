//go:build integration && rlseval

package rlseval

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// ADR-0010 evaluation requirement 4: legitimate cross-tenant access (admin reporting,
// reconciliation, background jobs). Two models are compared; neither is implemented in product
// code. Recommendation and auditing implications: rls-evaluation.md, §6.
//
//	model 1 "reporting": a dedicated role that is subject to RLS, with explicit policies written per
//	                     table and per command (here: SELECT only, all tenants).
//	model 2 "bypass":    a dedicated role with the BYPASSRLS attribute.

// TestCrossTenantExplicitPolicies: the reporting role reads every tenant's rows of a table that has
// a reporting policy, cannot write even where the privilege is granted (no write policy: default
// deny), and sees nothing in a table whose owner has not opted it in with a policy.
func TestCrossTenantExplicitPolicies(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	f.otherTable(t)
	reporting := f.pool(t, f.reporting, 1, nil)

	err := inTenant(ctx, reporting, "", func(tx pgx.Tx) error {
		got, err := visible(ctx, tx, "rlseval.items")
		if err != nil {
			return err
		}
		if !equalRows(got, rowsOf(tenantA, tenantB)) {
			t.Errorf("reporting role, items: %v, want every tenant's rows", got)
		}
		other, err := visible(ctx, tx, "rlseval.items_other")
		if err != nil {
			return err
		}
		if len(other) != 0 {
			t.Errorf("reporting role, table without a reporting policy: %v, want none (opt-in per table)", other)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// INSERT is granted, but there is no INSERT policy for this role.
	err = inTenant(ctx, reporting, "", func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO rlseval.items (tenant_id, label) VALUES ($1, 'x')`, tenantA)
		return err
	})
	wantSQLState(t, err, sqlStateInsufficientPrivilege)

	// UPDATE and DELETE are not granted at all.
	for _, stmt := range []string{`UPDATE rlseval.items SET label = 'x'`, `DELETE FROM rlseval.items`} {
		_, err := reporting.Exec(ctx, stmt)
		wantSQLState(t, err, sqlStateInsufficientPrivilege)
	}
}

// TestCrossTenantBypassRole: the BYPASSRLS role reads every tenant's rows of every table it has a
// privilege on, including a table with no policy for it, and writes rows for any tenant wherever a
// write privilege is granted: only table privileges limit it, and no policy is evaluated.
func TestCrossTenantBypassRole(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	f.otherTable(t)
	f.ownerExec(t, `GRANT INSERT ON rlseval.items_other TO `+ident(f.bypass))
	bypass := f.pool(t, f.bypass, 1, nil)

	err := inTenant(ctx, bypass, "", func(tx pgx.Tx) error {
		for _, table := range []string{"rlseval.items", "rlseval.items_other"} {
			got, err := visible(ctx, tx, table)
			if err != nil {
				return err
			}
			if !equalRows(got, rowsOf(tenantA, tenantB)) {
				t.Errorf("bypass role, %s: %v, want every tenant's rows", table, got)
			}
		}
		// A write for an arbitrary tenant, without any tenant context, succeeds.
		_, err := tx.Exec(ctx, `INSERT INTO rlseval.items_other (tenant_id, label) VALUES ($1, 'written-by-bypass')`, tenantD)
		return err
	})
	if err != nil {
		t.Fatalf("bypass role: %v", err)
	}
}

// otherTable creates rlseval.items_other: tenant-owned, RLS forced, a tenant policy for the
// application role only (no reporting policy), SELECT granted to both cross-tenant roles.
func (f *fixture) otherTable(t testing.TB) {
	t.Helper()
	f.ownerExec(t,
		`CREATE TABLE rlseval.items_other (tenant_id uuid NOT NULL, label text NOT NULL)`,
		`ALTER TABLE rlseval.items_other ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE rlseval.items_other FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY tenant_isolation ON rlseval.items_other TO `+ident(f.app)+` USING (`+tenantPolicy+`)`,
		`GRANT SELECT ON rlseval.items_other TO `+ident(f.reporting)+`, `+ident(f.bypass),
	)
	f.adminExec(t, `INSERT INTO rlseval.items_other SELECT tenant_id, label FROM rlseval.items`)
}
