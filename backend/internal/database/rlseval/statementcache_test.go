//go:build integration && rlseval

package rlseval

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgx statement caching and PostgreSQL plan caching with row-level security.
//
// sqlc-generated code sends each query as a fixed SQL string with parameters, so with pgx's default
// QueryExecModeCacheStatement every query becomes a prepared statement that is reused for the life
// of the connection, across transactions and therefore across tenants. After five executions
// PostgreSQL may switch a prepared statement to a cached generic plan. The policy reads the tenant
// through current_setting(), a STABLE function evaluated at execution time, so neither cache may
// carry one tenant's filter into another tenant's execution. This test checks that in every pgx
// execution mode, with plan_cache_mode forced to generic and to custom.

func TestStatementAndPlanCachesFollowTheContext(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	modes := []pgx.QueryExecMode{
		pgx.QueryExecModeCacheStatement, // pgx default
		pgx.QueryExecModeCacheDescribe,
		pgx.QueryExecModeDescribeExec,
		pgx.QueryExecModeExec,
		pgx.QueryExecModeSimpleProtocol,
	}
	sequence := []string{tenantA, tenantB, "", tenantA, tenantC, tenantB, tenantA, "", tenantB, tenantA, tenantB, tenantA}

	// The queries an sqlc repository would send: one relying on the policy alone, one with the
	// explicit tenant filter that application code keeps writing (AGENTS.md §10.7).
	queries := map[string]string{
		"policy only":     `SELECT tenant_id::text || '/' || label FROM rlseval.items WHERE label <> $1 ORDER BY 1`,
		"explicit filter": `SELECT tenant_id::text || '/' || label FROM rlseval.items WHERE tenant_id = $2 AND label <> $1 ORDER BY 1`,
	}

	for _, planMode := range []string{"force_generic_plan", "force_custom_plan"} {
		for _, mode := range modes {
			t.Run(fmt.Sprintf("%s/%s", planMode, mode), func(t *testing.T) {
				ctx := testContext(t)
				pool := f.pool(t, f.app, 1, func(c *pgxpool.Config) {
					c.ConnConfig.DefaultQueryExecMode = mode
					c.ConnConfig.RuntimeParams["plan_cache_mode"] = planMode
				})
				for name, query := range queries {
					for i, tenant := range sequence {
						err := inTenant(ctx, pool, tenant, func(tx pgx.Tx) error {
							args := []any{"none"}
							if name == "explicit filter" {
								args = append(args, nullableTenant(tenant))
							}
							rows, err := tx.Query(ctx, query, args...)
							if err != nil {
								return err
							}
							got, err := pgx.CollectRows(rows, pgx.RowTo[string])
							if err != nil {
								return err
							}
							if !equalRows(got, rowsOf(tenant)) {
								return fmt.Errorf("%s, execution %d, tenant %q: got %v, want %v", name, i, tenant, got, rowsOf(tenant))
							}
							return nil
						})
						if err != nil {
							t.Fatal(err)
						}
					}
				}
				if mode == pgx.QueryExecModeCacheStatement {
					assertPreparedPlans(ctx, t, pool, planMode, len(sequence))
				}
			})
		}
	}
}

// assertPreparedPlans proves that the cached statement was really reused: pg_prepared_statements
// shows the statement executed with generic plans (force_generic_plan) or custom plans.
func assertPreparedPlans(ctx context.Context, t *testing.T, pool *pgxpool.Pool, planMode string, executions int) {
	t.Helper()
	var generic, custom int64
	err := pool.QueryRow(ctx, `
		SELECT coalesce(sum(generic_plans), 0), coalesce(sum(custom_plans), 0)
		FROM pg_prepared_statements
		WHERE statement LIKE '%FROM rlseval.items WHERE%'
		  AND statement NOT LIKE '%pg_prepared_statements%'`).Scan(&generic, &custom)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: prepared statements used %d generic and %d custom plans", planMode, generic, custom)
	want := int64(2 * executions) // two queries, each executed once per tenant step
	if planMode == "force_generic_plan" && (generic != want || custom != 0) {
		t.Fatalf("generic/custom plans = %d/%d, want %d/0: the generic-plan path was not exercised", generic, custom, want)
	}
	if planMode == "force_custom_plan" && (custom != want || generic != 0) {
		t.Fatalf("generic/custom plans = %d/%d, want 0/%d", generic, custom, want)
	}
}

func nullableTenant(tenant string) any {
	if tenant == "" {
		return nil
	}
	return tenant
}
