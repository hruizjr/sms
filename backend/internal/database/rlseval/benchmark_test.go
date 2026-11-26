//go:build integration && rlseval

package rlseval

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Manual, INDICATIVE performance evaluation (ADR-0010 evaluation requirement 5; owner decision D:
// manual only, never a CI gate). Benchmarks run only with -bench, which CI never passes:
//
//	RUN_BENCHMARKS=1 bash scripts/rls-evaluation.sh
//
// Data: RLS_EVAL_BENCH_ROWS synthetic rows (default 1,000,000) spread evenly over 100 tenants, in
// two identical tables with a primary key (tenant_id, id) and an index (tenant_id, created_at):
//
//	bench_plain  no row-level security (baseline)
//	bench_rls    RLS enabled and forced, the fail-closed tenant policy
//
// Every iteration is one complete transaction, as a request would run it. Variants:
//
//	plain/filter              no RLS; explicit tenant filter; no set_config
//	plain/filter+set_config   as above plus set_config (isolates the cost of setting the context)
//	rls/filter+set_config     RLS plus the explicit filter (the proposed coding rule)
//	rls/policy-only           RLS with no explicit filter (relies on the policy alone)
//
// Results depend on the machine and are not capacity targets (rls-evaluation.md, §12).
func BenchmarkRLSOverhead(b *testing.B) {
	rowsWanted := 1_000_000
	if v := os.Getenv("RLS_EVAL_BENCH_ROWS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1000 {
			b.Fatalf("RLS_EVAL_BENCH_ROWS=%q: want an integer >= 1000", v)
		}
		rowsWanted = n
	}
	const tenants = 100

	f := newFixture(b)
	ctx := context.Background()
	f.ownerExec(b,
		`CREATE TABLE rlseval.bench_plain (
			tenant_id uuid NOT NULL, id uuid NOT NULL, created_at timestamptz NOT NULL, label text NOT NULL,
			PRIMARY KEY (tenant_id, id))`,
		`CREATE INDEX ix_bench_plain__tenant_id_created_at ON rlseval.bench_plain (tenant_id, created_at)`,
		`CREATE TABLE rlseval.bench_rls (LIKE rlseval.bench_plain INCLUDING ALL)`,
		`ALTER TABLE rlseval.bench_rls ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE rlseval.bench_rls FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY tenant_isolation ON rlseval.bench_rls TO `+ident(f.app)+
			` USING (`+tenantPolicy+`) WITH CHECK (`+tenantPolicy+`)`,
		`GRANT SELECT ON rlseval.bench_plain, rlseval.bench_rls TO `+ident(f.app),
	)
	// Deterministic synthetic data: tenant k is 00000000-0000-4000-8000-<k as 12 hex digits>.
	f.adminExec(b,
		`INSERT INTO rlseval.bench_plain
		 SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g % `+strconv.Itoa(tenants)+`), 12, '0'))::uuid,
		        gen_random_uuid(),
		        timestamptz '2026-01-01 00:00:00+00' + g * interval '1 second',
		        'synthetic-' || g
		 FROM generate_series(1, `+strconv.Itoa(rowsWanted)+`) AS g`,
		`INSERT INTO rlseval.bench_rls SELECT * FROM rlseval.bench_plain`,
		`ANALYZE rlseval.bench_plain`,
		`ANALYZE rlseval.bench_rls`,
	)
	tenant := "00000000-0000-4000-8000-00000000002a" // tenant 42
	var pointID string
	if err := f.admin.QueryRow(ctx, `SELECT id::text FROM rlseval.bench_plain WHERE tenant_id = $1 LIMIT 1`,
		tenant).Scan(&pointID); err != nil {
		b.Fatal(err)
	}
	app := f.pool(b, f.app, 1, nil) // pgx default exec mode (cached prepared statements)
	b.Logf("rows=%d tenants=%d rows/tenant=%d", rowsWanted, tenants, rowsWanted/tenants)

	queries := []struct {
		name, filtered, policyOnly string
		args                       func(tenant string) []any
	}{
		{
			name:       "point-lookup",
			filtered:   `SELECT label FROM rlseval.%s WHERE tenant_id = $1 AND id = $2`,
			policyOnly: `SELECT label FROM rlseval.%s WHERE id = $2 AND $1::uuid IS NOT NULL`,
			args:       func(tn string) []any { return []any{tn, pointID} },
		},
		{
			name:       "latest-50",
			filtered:   `SELECT label FROM rlseval.%s WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 50`,
			policyOnly: `SELECT label FROM rlseval.%s WHERE $1::uuid IS NOT NULL ORDER BY created_at DESC LIMIT 50`,
			args:       func(tn string) []any { return []any{tn} },
		},
		{
			name:       "count-tenant",
			filtered:   `SELECT count(*)::text FROM rlseval.%s WHERE tenant_id = $1`,
			policyOnly: `SELECT count(*)::text FROM rlseval.%s WHERE $1::uuid IS NOT NULL`,
			args:       func(tn string) []any { return []any{tn} },
		},
	}

	for _, q := range queries {
		variants := []struct {
			name, sql  string
			setContext bool
		}{
			{"plain/filter", sprintfTable(q.filtered, "bench_plain"), false},
			{"plain/filter+set_config", sprintfTable(q.filtered, "bench_plain"), true},
			{"rls/filter+set_config", sprintfTable(q.filtered, "bench_rls"), true},
			{"rls/policy-only", sprintfTable(q.policyOnly, "bench_rls"), true},
		}
		for _, v := range variants {
			logPlan(ctx, b, app, q.name+" "+v.name, v.sql, v.setContext, q.args(tenant))
			b.Run(q.name+"/"+v.name, func(b *testing.B) {
				args := q.args(tenant)
				for b.Loop() {
					err := pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
						if v.setContext {
							if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenant); err != nil {
								return err
							}
						}
						rows, err := tx.Query(ctx, v.sql, args...)
						if err != nil {
							return err
						}
						got, err := pgx.CollectRows(rows, pgx.RowTo[string])
						if err != nil {
							return err
						}
						if len(got) == 0 {
							b.Fatal("query returned no rows")
						}
						return nil
					})
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// logPlan records EXPLAIN (ANALYZE, BUFFERS) for one variant, executed as the application role.
func logPlan(ctx context.Context, b *testing.B, pool *pgxpool.Pool, name, sql string, setContext bool, args []any) {
	b.Helper()
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if setContext {
			if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, args[0]); err != nil {
				return err
			}
		}
		rows, err := tx.Query(ctx, `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY OFF) `+sql, args...)
		if err != nil {
			return err
		}
		lines, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		b.Logf("PLAN %s\n    %s", name, strings.Join(lines, "\n    "))
		return nil
	})
	if err != nil {
		b.Fatalf("explain %s: %v", name, err)
	}
}

func sprintfTable(format, table string) string { return strings.Replace(format, "%s", table, 1) }
