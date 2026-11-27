//go:build integration

package rlsguard_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/database/rlsguard"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func openAdmin(t *testing.T, ctx context.Context) *database.DB {
	t.Helper()
	db, err := database.Open(ctx, dbtest.NewMigratedDatabase(t).Admin, "sms-rlsguard-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

// TestIntegrationMigratedSchemaPassesGuard is the CI guard (owner decision RLS-1, testing standards
// §6): after every migration is applied, no tenant-owned table may lack row-level security
// (enabled and forced) or its policy. It fails the build otherwise.
func TestIntegrationMigratedSchemaPassesGuard(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	violations, err := rlsguard.Check(ctx, openAdmin(t, ctx).Pool())
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	for _, v := range violations {
		t.Errorf("unprotected tenant-owned table %s", v)
	}
}

// TestIntegrationGuardDetectsUnprotectedTables adds deliberately unprotected tenant-owned tables to a
// migrated database, including tables whose policies widen or change the approved policy, and
// checks that the guard reports each one, with exactly the expected problems, and only those.
func TestIntegrationGuardDetectsUnprotectedTables(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	db := openAdmin(t, ctx)
	const policy = `(tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)`
	for _, sql := range []string{
		`CREATE SCHEMA guard_test`,
		// Protected correctly: not reported.
		`CREATE TABLE guard_test.protected (tenant_id uuid NOT NULL)`,
		`ALTER TABLE guard_test.protected ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE guard_test.protected FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY p ON guard_test.protected USING ` + policy + ` WITH CHECK ` + policy,
		// No row-level security at all.
		`CREATE TABLE public.no_rls (tenant_id uuid NOT NULL)`,
		// Enabled but not forced (the owner would bypass the policy).
		`CREATE TABLE public.not_forced (tenant_id uuid NOT NULL)`,
		`ALTER TABLE public.not_forced ENABLE ROW LEVEL SECURITY`,
		`CREATE POLICY p ON public.not_forced USING ` + policy + ` WITH CHECK ` + policy,
		// Enabled and forced, but no policy.
		`CREATE TABLE public.no_policy (tenant_id uuid NOT NULL)`,
		`ALTER TABLE public.no_policy ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.no_policy FORCE ROW LEVEL SECURITY`,
		// A policy for SELECT only: writes are not checked.
		`CREATE TABLE public.select_only (tenant_id uuid NOT NULL)`,
		`ALTER TABLE public.select_only ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.select_only FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY p ON public.select_only FOR SELECT USING ` + policy,
		// A policy that does not use the tenant setting.
		`CREATE TABLE public.wrong_policy (tenant_id uuid NOT NULL)`,
		`ALTER TABLE public.wrong_policy ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.wrong_policy FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY p ON public.wrong_policy USING (true) WITH CHECK (true)`,
		// A partitioned table protected on the parent but with an unprotected partition, which can
		// be queried directly.
		`CREATE TABLE public.parted (tenant_id uuid NOT NULL) PARTITION BY HASH (tenant_id)`,
		`ALTER TABLE public.parted ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.parted FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY p ON public.parted USING ` + policy + ` WITH CHECK ` + policy,
		`CREATE TABLE public.parted_0 PARTITION OF public.parted FOR VALUES WITH (MODULUS 1, REMAINDER 0)`,
		// Not tenant-owned (no tenant_id column): not checked.
		`CREATE TABLE public.platform_lookup (code text PRIMARY KEY)`,
	} {
		if _, err := db.Pool().Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	// Policies that widen or change the approved policy. Each table also has RLS enabled and forced.
	for _, tc := range []struct{ table, policies string }{
		// The approved policy plus a permissive SELECT policy: PostgreSQL ORs permissive policies, so
		// every tenant's rows become readable.
		{"extra_permissive", `CREATE POLICY p ON %[1]s USING ` + policy + ` WITH CHECK ` + policy + `;
			CREATE POLICY open_read ON %[1]s FOR SELECT USING (true)`},
		// The approved policy plus a permissive INSERT policy: any tenant's rows can be written.
		{"extra_permissive_insert", `CREATE POLICY p ON %[1]s USING ` + policy + ` WITH CHECK ` + policy + `;
			CREATE POLICY open_insert ON %[1]s FOR INSERT WITH CHECK (true)`},
		// The tenant condition weakened with OR true: it still mentions app.tenant_id.
		{"or_true", `CREATE POLICY p ON %[1]s USING (` + policy + ` OR true) WITH CHECK (` + policy + ` OR true)`},
		// The right USING, but writes are not checked.
		{"open_with_check", `CREATE POLICY p ON %[1]s USING ` + policy + ` WITH CHECK (true)`},
		// No explicit WITH CHECK.
		{"no_with_check", `CREATE POLICY p ON %[1]s USING ` + policy},
		// Without NULLIF, an empty setting raises an error instead of returning no rows (RLS evaluation §5).
		{"no_nullif", `CREATE POLICY p ON %[1]s USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
			WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid)`},
		// The approved expressions, but only for one role: every other role sees no rows.
		{"one_role", `CREATE POLICY p ON %[1]s TO sms_migrator USING ` + policy + ` WITH CHECK ` + policy},
		// The approved policy plus a restrictive policy: it cannot widen access, but the guard fails
		// closed on any policy that is not approved.
		{"extra_restrictive", `CREATE POLICY p ON %[1]s USING ` + policy + ` WITH CHECK ` + policy + `;
			CREATE POLICY narrow ON %[1]s AS RESTRICTIVE USING (true)`},
	} {
		table := "public." + tc.table
		for _, sql := range []string{
			`CREATE TABLE ` + table + ` (tenant_id uuid NOT NULL)`,
			`ALTER TABLE ` + table + ` ENABLE ROW LEVEL SECURITY`,
			`ALTER TABLE ` + table + ` FORCE ROW LEVEL SECURITY`,
			fmt.Sprintf(tc.policies, table),
		} {
			if _, err := db.Pool().Exec(ctx, sql); err != nil {
				t.Fatalf("%s: %v", sql, err)
			}
		}
	}

	violations, err := rlsguard.Check(ctx, db.Pool())
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	got := map[string][]string{}
	for _, v := range violations {
		got[v.Table] = v.Problems
	}
	// The approved expression as PostgreSQL 18 prints it.
	const approved = `(tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)`
	const noApproved = "has no approved tenant-isolation policy"
	notApproved := func(name string, reasons ...string) string {
		return fmt.Sprintf("policy %q is not the approved tenant-isolation policy (%s)", name, strings.Join(reasons, "; "))
	}
	want := map[string][]string{
		"public.no_rls":     {"row-level security is not enabled", "row-level security is not forced", "has no policy"},
		"public.not_forced": {"row-level security is not forced"},
		"public.no_policy":  {"has no policy"},
		"public.select_only": {
			notApproved("p", "not for ALL commands: SELECT", "no WITH CHECK expression"), noApproved},
		"public.wrong_policy": {notApproved("p", "USING is true", "WITH CHECK is true"), noApproved},
		"public.parted_0":     {"row-level security is not enabled", "row-level security is not forced", "has no policy"},
		"public.extra_permissive": {
			notApproved("open_read", "not for ALL commands: SELECT", "USING is true", "no WITH CHECK expression")},
		"public.extra_permissive_insert": {
			notApproved("open_insert", "not for ALL commands: INSERT", "no USING expression", "WITH CHECK is true")},
		"public.or_true":         {notApproved("p", "USING is ("+approved+" OR true)", "WITH CHECK is ("+approved+" OR true)"), noApproved},
		"public.open_with_check": {notApproved("p", "WITH CHECK is true"), noApproved},
		"public.no_with_check":   {notApproved("p", "no WITH CHECK expression"), noApproved},
		"public.no_nullif": {notApproved("p",
			"USING is (tenant_id = (current_setting('app.tenant_id'::text, true))::uuid)",
			"WITH CHECK is (tenant_id = (current_setting('app.tenant_id'::text, true))::uuid)"), noApproved},
		"public.one_role":          {notApproved("p", "applies only to specific roles"), noApproved},
		"public.extra_restrictive": {notApproved("narrow", "restrictive", "USING is true", "no WITH CHECK expression")},
	}
	if len(got) != len(want) {
		t.Errorf("Check() reported %d tables, want %d: %v", len(got), len(want), violations)
	}
	for table, problems := range want {
		if !slices.Equal(got[table], problems) {
			t.Errorf("%s: problems %q, want %q", table, got[table], problems)
		}
	}
}
