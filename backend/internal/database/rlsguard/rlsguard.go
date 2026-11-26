// Package rlsguard is the row-level security catalog guard (owner decision RLS-1; testing
// standards §6; decision log §18).
//
// After the migrations are applied, Check lists every tenant-owned table that is not protected as
// ADR-0010 requires. A tenant-owned table is any table (ordinary, partitioned, or a partition) with
// a tenant_id column in a schema other than the PostgreSQL system schemas. Each must have:
//
//   - row-level security enabled;
//   - row-level security forced (so the table owner is subject to it as well);
//   - the approved tenant-isolation policy (owner decision RLS-2): PERMISSIVE, FOR ALL commands, TO
//     PUBLIC, with exactly the approved expression (below) in both USING and WITH CHECK;
//   - no other policy. The guard fails closed: any other policy is reported, whether it could widen
//     access (an extra permissive policy, or the tenant expression weakened with OR) or only
//     narrows it (a restrictive policy). PostgreSQL combines permissive policies with OR, so a
//     single extra permissive policy can expose every tenant's rows.
//
// The approved expression is
//
//	tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
//
// The expressions are compared as PostgreSQL prints them (pg_get_expr), so formatting differences
// in the migration do not matter, but any change of meaning does. If a PostgreSQL upgrade prints
// the approved expression differently, the guard reports the approved policy too (it fails
// closed) and TestIntegrationGuardDetectsUnprotectedTables fails; update approvedExpression then.
//
// CI runs Check against a freshly migrated database (TestIntegrationMigratedSchemaPassesGuard) and
// fails on any finding. The platform-owned tenants table has no tenant_id column and is therefore
// not checked.
package rlsguard

import (
	"context"
	"fmt"
	"strings"

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
	Problems []string // what is missing or not allowed
}

func (v Violation) String() string { return fmt.Sprintf("%s: %v", v.Table, v.Problems) }

// Querier is satisfied by *pgx.Conn, *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// tenantSetting is database.TenantSetting; it is repeated here so that this package does not
// depend on the connection pool package. TestTenantSettingMatches keeps them equal.
const tenantSetting = "app.tenant_id"

// approvedExpression is the approved RLS-2 policy expression (package documentation) as
// PostgreSQL 18 prints it.
const approvedExpression = `(tenant_id = (NULLIF(current_setting('` + tenantSetting + `'::text, true), ''::text))::uuid)`

// query lists each tenant-owned table with each of its policies (one row per policy; one row with
// NULL policy columns for a table without any policy).
const query = `
SELECT format('%I.%I', n.nspname, c.relname),
       c.relrowsecurity,
       c.relforcerowsecurity,
       p.polname::text,
       p.polcmd::text,
       p.polpermissive,
       p.polroles = '{0}'::oid[],
       pg_get_expr(p.polqual, p.polrelid),
       pg_get_expr(p.polwithcheck, p.polrelid)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_policy p ON p.polrelid = c.oid
WHERE c.relkind IN ('r', 'p')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
  AND n.nspname NOT LIKE 'pg\_temp\_%'
  AND n.nspname NOT LIKE 'pg\_toast\_temp\_%'
  AND EXISTS (SELECT FROM pg_attribute a
              WHERE a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped)
ORDER BY 1, 4`

// policyRow is one row of query. The policy columns are NULL when the table has no policy.
type policyRow struct {
	Table      string
	Enabled    bool
	Forced     bool
	Name       *string
	Command    *string // pg_policy.polcmd: r, a, w, d or *
	Permissive *bool
	ToPublic   *bool
	Using      *string
	WithCheck  *string
}

// Check returns the unprotected tenant-owned tables of the connected database, in name order,
// excluding those in Exceptions. An empty result means every tenant-owned table is protected.
func Check(ctx context.Context, q Querier) ([]Violation, error) {
	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("rls catalog guard: %w", err)
	}
	all, err := pgx.CollectRows(rows, pgx.RowToStructByPos[policyRow])
	if err != nil {
		return nil, fmt.Errorf("rls catalog guard: %w", err)
	}
	var violations []Violation
	for start := 0; start < len(all); {
		end := start + 1
		for end < len(all) && all[end].Table == all[start].Table {
			end++
		}
		table := all[start].Table
		if _, exempt := Exceptions[table]; !exempt {
			if problems := tableProblems(all[start:end]); len(problems) > 0 {
				violations = append(violations, Violation{Table: table, Problems: problems})
			}
		}
		start = end
	}
	return violations, nil
}

// tableProblems evaluates the rows of one table (all with the same Table).
func tableProblems(rows []policyRow) []string {
	var problems []string
	if !rows[0].Enabled {
		problems = append(problems, "row-level security is not enabled")
	}
	if !rows[0].Forced {
		problems = append(problems, "row-level security is not forced")
	}
	if rows[0].Name == nil {
		return append(problems, "has no policy")
	}
	approved := false
	for _, p := range rows {
		reasons := policyDeviations(p)
		if len(reasons) == 0 {
			approved = true
			continue
		}
		problems = append(problems, fmt.Sprintf("policy %q is not the approved tenant-isolation policy (%s)",
			*p.Name, strings.Join(reasons, "; ")))
	}
	if !approved {
		problems = append(problems, "has no approved tenant-isolation policy")
	}
	return problems
}

// policyDeviations lists how p differs from the approved tenant-isolation policy; none means p is
// the approved policy. Unknown or missing values count as deviations (fail closed).
func policyDeviations(p policyRow) []string {
	var reasons []string
	if p.Permissive == nil || !*p.Permissive {
		reasons = append(reasons, "restrictive")
	}
	if p.Command == nil || *p.Command != "*" {
		reasons = append(reasons, "not for ALL commands: "+commandName(p.Command))
	}
	if p.ToPublic == nil || !*p.ToPublic {
		reasons = append(reasons, "applies only to specific roles")
	}
	reasons = appendExpressionDeviation(reasons, "USING", p.Using)
	reasons = appendExpressionDeviation(reasons, "WITH CHECK", p.WithCheck)
	return reasons
}

func appendExpressionDeviation(reasons []string, clause string, expr *string) []string {
	switch {
	case expr == nil:
		return append(reasons, "no "+clause+" expression")
	case *expr != approvedExpression:
		return append(reasons, clause+" is "+*expr)
	}
	return reasons
}

func commandName(cmd *string) string {
	if cmd == nil {
		return "unknown"
	}
	switch *cmd {
	case "r":
		return "SELECT"
	case "a":
		return "INSERT"
	case "w":
		return "UPDATE"
	case "d":
		return "DELETE"
	}
	return *cmd
}
