// Package rlsguard is the row-level security catalog guard (owner decision RLS-1; testing
// standards §6; decision log §18).
//
// After the migrations are applied, Check lists every tenant-owned table that is not protected as
// ADR-0010 requires. A tenant-owned table is any table (ordinary, partitioned, or a partition) with
// a tenant_id column in a schema other than the PostgreSQL system schemas. Each must have:
//
//   - row-level security enabled;
//   - row-level security forced (so the table owner is subject to it as well);
//   - at least one policy, and a policy that applies to ALL commands with both a USING and a WITH
//     CHECK expression that read the app.tenant_id setting (database.TenantSetting).
//
// CI runs Check against a freshly migrated database (TestIntegrationMigratedSchemaPassesGuard) and
// fails on any finding. The platform-owned tenants table has no tenant_id column and is therefore
// not checked.
package rlsguard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Exceptions lists tenant-owned tables that are exempt from the guard, keyed by schema-qualified
// name, with the owner-approved reason. A table may be added only with the project owner's
// approval, recorded in the decision log (testing standards §6); deferred high-volume data-plane
// tables (RLS-5) belong here rather than being skipped silently. There are no exceptions.
var Exceptions = map[string]string{}

// Violation is one unprotected tenant-owned table.
type Violation struct {
	Table    string   // schema-qualified name
	Problems []string // what is missing
}

func (v Violation) String() string { return fmt.Sprintf("%s: %v", v.Table, v.Problems) }

// Querier is satisfied by *pgx.Conn, *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// query lists each tenant-owned table with its findings. The policy check looks for a policy for
// ALL commands whose USING and WITH CHECK expressions both mention app.tenant_id.
const query = `
SELECT format('%I.%I', n.nspname, c.relname) AS table_name,
       array_remove(ARRAY[
           CASE WHEN NOT c.relrowsecurity THEN 'row-level security is not enabled' END,
           CASE WHEN NOT c.relforcerowsecurity THEN 'row-level security is not forced' END,
           CASE WHEN NOT EXISTS (SELECT FROM pg_policy p WHERE p.polrelid = c.oid)
                THEN 'has no policy' END,
           CASE WHEN EXISTS (SELECT FROM pg_policy p WHERE p.polrelid = c.oid)
                 AND NOT EXISTS (
                     SELECT FROM pg_policy p
                     WHERE p.polrelid = c.oid
                       AND p.polcmd = '*'
                       AND p.polqual IS NOT NULL
                       AND p.polwithcheck IS NOT NULL
                       AND strpos(pg_get_expr(p.polqual, p.polrelid), $1) > 0
                       AND strpos(pg_get_expr(p.polwithcheck, p.polrelid), $1) > 0)
                THEN 'has no policy for ALL commands with USING and WITH CHECK on ' || $2 END
       ], NULL) AS problems
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'p')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
  AND n.nspname NOT LIKE 'pg\_temp\_%'
  AND n.nspname NOT LIKE 'pg\_toast\_temp\_%'
  AND EXISTS (SELECT FROM pg_attribute a
              WHERE a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped)
ORDER BY 1`

// tenantSetting is database.TenantSetting; it is repeated here so that this package does not
// depend on the connection pool package. TestTenantSettingMatches keeps them equal.
const tenantSetting = "app.tenant_id"

// Check returns the unprotected tenant-owned tables of the connected database, in name order,
// excluding those in Exceptions. An empty result means every tenant-owned table is protected.
func Check(ctx context.Context, q Querier) ([]Violation, error) {
	rows, err := q.Query(ctx, query, "'"+tenantSetting+"'", tenantSetting)
	if err != nil {
		return nil, fmt.Errorf("rls catalog guard: %w", err)
	}
	all, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Violation])
	if err != nil {
		return nil, fmt.Errorf("rls catalog guard: %w", err)
	}
	var violations []Violation
	for _, v := range all {
		if _, exempt := Exceptions[v.Table]; exempt || len(v.Problems) == 0 {
			continue
		}
		violations = append(violations, v)
	}
	return violations, nil
}
