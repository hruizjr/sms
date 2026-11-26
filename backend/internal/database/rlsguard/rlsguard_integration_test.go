//go:build integration

package rlsguard_test

import (
	"context"
	"slices"
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
// migrated database and checks that the guard reports each one, and only those.
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

	violations, err := rlsguard.Check(ctx, db.Pool())
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}
	got := map[string][]string{}
	for _, v := range violations {
		got[v.Table] = v.Problems
	}
	const noAllPolicy = "has no policy for ALL commands with USING and WITH CHECK on app.tenant_id"
	want := map[string][]string{
		"public.no_rls":       {"row-level security is not enabled", "row-level security is not forced", "has no policy"},
		"public.not_forced":   {"row-level security is not forced"},
		"public.no_policy":    {"has no policy"},
		"public.select_only":  {noAllPolicy},
		"public.wrong_policy": {noAllPolicy},
		"public.parted_0":     {"row-level security is not enabled", "row-level security is not forced", "has no policy"},
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
