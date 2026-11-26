//go:build integration && rlseval

package rlseval

import (
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Documented PostgreSQL row-level security pitfalls, each shown with its mitigation.

// TestUniqueConstraintRevealsOtherTenants: a unique constraint is checked against every row, not
// only the visible ones. With a global UNIQUE (code), tenant B learns that some other tenant already
// uses a code (an existence side channel). PostgreSQL does omit the usual "Key (code)=(...) already
// exists" detail when row security applies to the caller, so the error does not repeat other
// tenants' values. A unique constraint that leads with tenant_id removes the side channel.
func TestUniqueConstraintRevealsOtherTenants(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	for _, c := range []struct{ table, unique string }{
		{"codes_global", "UNIQUE (code)"},
		{"codes_tenant", "UNIQUE (tenant_id, code)"},
	} {
		f.ownerExec(t,
			`CREATE TABLE rlseval.`+c.table+` (tenant_id uuid NOT NULL, code text NOT NULL, `+c.unique+`)`,
			`ALTER TABLE rlseval.`+c.table+` ENABLE ROW LEVEL SECURITY`,
			`ALTER TABLE rlseval.`+c.table+` FORCE ROW LEVEL SECURITY`,
			`CREATE POLICY tenant_isolation ON rlseval.`+c.table+` TO `+ident(f.app)+
				` USING (`+tenantPolicy+`) WITH CHECK (`+tenantPolicy+`)`,
			`GRANT SELECT, INSERT ON rlseval.`+c.table+` TO `+ident(f.app),
		)
		f.adminExec(t, `INSERT INTO rlseval.`+c.table+` VALUES ('`+tenantA+`', 'SYNTHETIC-CODE-1')`)
	}
	app := f.pool(t, f.app, 1, nil)
	insert := func(table string) error {
		return inTenant(ctx, app, tenantB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO rlseval.`+table+` VALUES ($1, 'SYNTHETIC-CODE-1')`, tenantB)
			return err
		})
	}

	pgErr := asPgError(t, insert("codes_global"))
	if pgErr.Code != sqlStateUniqueViolation {
		t.Fatalf("SQLSTATE = %s (%s), want %s", pgErr.Code, pgErr.Message, sqlStateUniqueViolation)
	}
	t.Logf("tenant B, global unique constraint: %s (constraint %s, detail %q)", pgErr.Message, pgErr.ConstraintName, pgErr.Detail)
	if strings.Contains(pgErr.Detail, "SYNTHETIC-CODE-1") {
		t.Errorf("detail = %q: the key value of another tenant's row was disclosed", pgErr.Detail)
	}
	if err := insert("codes_tenant"); err != nil {
		t.Fatalf("tenant-scoped unique constraint: %v", err)
	}
}

// TestForeignKeyChecksIgnoreRLS: referential integrity checks are not subject to row-level
// security. With a foreign key on parent_id alone, tenant B can attach a child to tenant A's
// parent (a cross-tenant reference), and the error for a missing parent tells B whether a parent
// ID exists in another tenant. A composite foreign key (tenant_id, parent_id) makes the reference
// impossible and the two cases indistinguishable.
func TestForeignKeyChecksIgnoreRLS(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	const parentOfA = "aaaaaaaa-1111-4000-8000-00000000000a"
	const noParent = "eeeeeeee-1111-4000-8000-00000000000e"

	stmts := []string{
		`CREATE TABLE rlseval.parents (tenant_id uuid NOT NULL, id uuid NOT NULL PRIMARY KEY, UNIQUE (tenant_id, id))`,
		`CREATE TABLE rlseval.children_simple_fk (tenant_id uuid NOT NULL,
			parent_id uuid NOT NULL REFERENCES rlseval.parents (id))`,
		`CREATE TABLE rlseval.children_tenant_fk (tenant_id uuid NOT NULL, parent_id uuid NOT NULL,
			FOREIGN KEY (tenant_id, parent_id) REFERENCES rlseval.parents (tenant_id, id))`,
	}
	for _, table := range []string{"parents", "children_simple_fk", "children_tenant_fk"} {
		stmts = append(stmts,
			`ALTER TABLE rlseval.`+table+` ENABLE ROW LEVEL SECURITY`,
			`ALTER TABLE rlseval.`+table+` FORCE ROW LEVEL SECURITY`,
			`CREATE POLICY tenant_isolation ON rlseval.`+table+` TO `+ident(f.app)+
				` USING (`+tenantPolicy+`) WITH CHECK (`+tenantPolicy+`)`,
			`GRANT SELECT, INSERT ON rlseval.`+table+` TO `+ident(f.app),
		)
	}
	f.ownerExec(t, stmts...)
	f.adminExec(t, `INSERT INTO rlseval.parents VALUES ('`+tenantA+`', '`+parentOfA+`')`)
	app := f.pool(t, f.app, 1, nil)

	insertChild := func(table, parent string) error {
		return inTenant(ctx, app, tenantB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO rlseval.`+table+` VALUES ($1, $2)`, tenantB, parent)
			return err
		})
	}

	// Simple foreign key: tenant B cannot see A's parent, yet can reference it...
	if err := insertChild("children_simple_fk", parentOfA); err != nil {
		t.Fatalf("simple FK, A's parent: %v (expected the demonstrated cross-tenant reference to succeed)", err)
	}
	// ...and a non-existent parent fails differently, which reveals that A's parent exists.
	wantSQLState(t, insertChild("children_simple_fk", noParent), sqlStateForeignKeyViolation)

	// Composite foreign key: both fail the same way.
	wantSQLState(t, insertChild("children_tenant_fk", parentOfA), sqlStateForeignKeyViolation)
	wantSQLState(t, insertChild("children_tenant_fk", noParent), sqlStateForeignKeyViolation)
}

// TestViewsAndSecurityDefinerFunctionsRunAsOwner: a view checks the base table's row security as
// the view's owner, and a SECURITY DEFINER function runs as its owner. If the owner bypasses RLS
// (an owner of an unforced table), both expose every tenant to the application role. A view with
// security_invoker = true and a SECURITY INVOKER function are filtered for the caller. On a forced
// table the owner itself has no policy, so the owner-checked view returns nothing (default deny)
// instead of leaking.
func TestViewsAndSecurityDefinerFunctionsRunAsOwner(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	f.noForceTable(t)
	f.ownerExec(t,
		`CREATE VIEW rlseval.v_noforce_owner AS SELECT tenant_id, label FROM rlseval.items_noforce`,
		`CREATE VIEW rlseval.v_noforce_invoker WITH (security_invoker = true) AS
			SELECT tenant_id, label FROM rlseval.items_noforce`,
		`CREATE VIEW rlseval.v_forced_owner AS SELECT tenant_id, label FROM rlseval.items`,
		`CREATE FUNCTION rlseval.f_definer() RETURNS TABLE (tenant_id uuid, label text)
			LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog
			AS 'SELECT tenant_id, label FROM rlseval.items_noforce'`,
		`CREATE FUNCTION rlseval.f_invoker() RETURNS TABLE (tenant_id uuid, label text)
			LANGUAGE sql STABLE SECURITY INVOKER SET search_path = pg_catalog
			AS 'SELECT tenant_id, label FROM rlseval.items_noforce'`,
		`GRANT SELECT ON rlseval.v_noforce_owner, rlseval.v_noforce_invoker, rlseval.v_forced_owner TO `+ident(f.app),
	)
	app := f.pool(t, f.app, 1, nil)

	cases := []struct {
		relation string
		want     []string
	}{
		{"rlseval.v_noforce_owner", rowsOf(tenantA, tenantB)}, // leak
		{"rlseval.f_definer()", rowsOf(tenantA, tenantB)},     // leak
		{"rlseval.v_noforce_invoker", rowsOf(tenantA)},
		{"rlseval.f_invoker()", rowsOf(tenantA)},
		{"rlseval.v_forced_owner", nil}, // default deny for the owner
	}
	err := inTenant(ctx, app, tenantA, func(tx pgx.Tx) error {
		for _, c := range cases {
			got, err := visible(ctx, tx, c.relation)
			if err != nil {
				return err
			}
			if !equalRows(got, c.want) {
				t.Errorf("%s as the application role in tenant A: %v, want %v", c.relation, got, c.want)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestCatalogGuardFindsUnprotectedTables shows that a future CI check is feasible: one catalog query
// lists every table with a tenant_id column that does not have row security both enabled and
// forced, or has no policy. (A proposal only; nothing in CI uses it yet.)
func TestCatalogGuardFindsUnprotectedTables(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := testContext(t)
	f.noForceTable(t)
	f.ownerExec(t,
		`CREATE TABLE rlseval.items_no_rls (tenant_id uuid NOT NULL, label text NOT NULL)`,
		`CREATE TABLE rlseval.items_no_policy (tenant_id uuid NOT NULL, label text NOT NULL)`,
		`ALTER TABLE rlseval.items_no_policy ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE rlseval.items_no_policy FORCE ROW LEVEL SECURITY`,
		`CREATE TABLE rlseval.not_tenant_owned (code text PRIMARY KEY)`,
	)

	rows, err := f.admin.Query(ctx, `
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'rlseval'
		  AND c.relkind IN ('r', 'p')
		  AND EXISTS (SELECT 1 FROM pg_attribute a
		              WHERE a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped)
		  AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity
		       OR NOT EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid = c.oid))
		ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"items_no_policy", "items_no_rls", "items_noforce"}
	if !slices.Equal(got, want) {
		t.Fatalf("unprotected tenant tables = %v, want %v", got, want)
	}
}
