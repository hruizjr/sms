//go:build integration

package migration_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbroles"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/migration"
	"github.com/hruizjr/sms/backend/migrations"
)

// Versions of the embedded migrations.
const (
	baselineVersion  = 20261125041208 // migrations/20261125041208_baseline.sql
	tenantsVersion   = 20261126175603 // migrations/20261126175603_create_tenants.sql
	messagingVersion = 20261127150000 // migrations/20261127150000_create_messaging.sql
	portalVersion    = 20261127200000 // migrations/20261127200000_create_portal_accounts.sql
	latestVersion    = 20261128120000 // migrations/20261128120000_create_security_audit_and_sign_in_limits.sql
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// sqlDB opens a database/sql handle on cfg backed by the shared pool package, as cmd/migrate does.
func sqlDB(t *testing.T, cfg config.DatabaseConfig) (*database.DB, *sql.DB) {
	t.Helper()
	db, err := database.Open(context.Background(), cfg, "sms-migration-test")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	sdb := stdlib.OpenDBFromPool(db.Pool())
	t.Cleanup(func() {
		_ = sdb.Close()
		db.Close()
	})
	return db, sdb
}

func newRunner(t *testing.T, sdb *sql.DB, fsys fstest.MapFS) *migration.Runner {
	t.Helper()
	var r *migration.Runner
	var err error
	if fsys == nil {
		r, err = migration.New(sdb, migrations.FS)
	} else {
		r, err = migration.New(sdb, fsys)
	}
	if err != nil {
		t.Fatalf("migration.New() error: %v", err)
	}
	return r
}

// TestIntegrationEmbeddedMigrations applies, rolls back and re-applies the real migrations as the
// migration role, as the migrate command does, and checks the resulting tables and their owner.
func TestIntegrationEmbeddedMigrations(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	db, sdb := sqlDB(t, dbtest.RoleConfig(t, dbtest.NewDatabase(t), dbroles.Migrator))
	runner := newRunner(t, sdb, nil)

	statuses, err := runner.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if len(statuses) == 0 || statuses[0].State != goose.StatePending {
		t.Fatalf("a fresh database must have every migration pending, got %+v", statuses)
	}

	results, err := runner.Up(ctx)
	if err != nil {
		t.Fatalf("Up() error: %v", err)
	}
	if len(results) != len(statuses) {
		t.Errorf("Up() applied %d migrations, want %d", len(results), len(statuses))
	}
	assertVersion(t, ctx, runner, latestVersion)

	statuses, err = runner.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	for _, s := range statuses {
		if s.State != goose.StateApplied || s.AppliedAt.IsZero() {
			t.Errorf("after Up(), %s is %s, want applied", s.Source.Path, s.State)
		}
	}

	// The complete list of tables, each owned by the migration role. Update this list deliberately
	// with every migration that adds or removes a table.
	want := []string{
		"public.api_keys sms_migrator", "public.dispatch_jobs sms_migrator", "public.goose_db_version sms_migrator",
		"public.idempotency_keys sms_migrator", "public.message_bodies sms_migrator", "public.message_receipts sms_migrator",
		"public.message_status_history sms_migrator", "public.messages sms_migrator",
		"public.platform_security_events sms_migrator", "public.portal_login_directory sms_migrator",
		"public.portal_sessions sms_migrator", "public.portal_sign_in_rate_limits sms_migrator", "public.portal_users sms_migrator",
		"public.provider_receipts sms_migrator", "public.security_audit_events sms_migrator", "public.sender_ids sms_migrator",
		"public.submission_attempts sms_migrator", "public.tenants sms_migrator",
	}
	group6 := []string{"public.platform_security_events sms_migrator", "public.portal_sign_in_rate_limits sms_migrator",
		"public.security_audit_events sms_migrator"}
	afterPortal := slices.DeleteFunc(slices.Clone(want), func(s string) bool { return slices.Contains(group6, s) })
	afterMessaging := slices.DeleteFunc(slices.Clone(afterPortal), func(s string) bool { return strings.HasPrefix(s, "public.portal_") })
	afterTenants := []string{"public.goose_db_version sms_migrator", "public.tenants sms_migrator"}
	if got := tablesAndOwners(t, ctx, db); !slices.Equal(got, want) {
		t.Errorf("tables after migrating = %v, want %v", got, want)
	}

	// Up is idempotent.
	again, err := runner.Up(ctx)
	if err != nil || len(again) != 0 {
		t.Errorf("second Up() = %d results, %v; want 0, nil", len(again), err)
	}

	// Down rolls back one migration at a time (local development and tests only): first the
	// security audit and rate-limit tables, then the portal tables, then the messaging tables, then
	// the tenants table, then the baseline. Rolling back past zero is reported.
	if _, err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	assertVersion(t, ctx, runner, portalVersion)
	if got := tablesAndOwners(t, ctx, db); !slices.Equal(got, afterPortal) {
		t.Errorf("tables after rolling back the security audit migration = %v, want %v", got, afterPortal)
	}
	if _, err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	assertVersion(t, ctx, runner, messagingVersion)
	if got := tablesAndOwners(t, ctx, db); !slices.Equal(got, afterMessaging) {
		t.Errorf("tables after rolling back the portal migration = %v, want %v", got, afterMessaging)
	}
	if _, err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	assertVersion(t, ctx, runner, tenantsVersion)
	if got := tablesAndOwners(t, ctx, db); !slices.Equal(got, afterTenants) {
		t.Errorf("tables after rolling back the messaging migration = %v, want %v", got, afterTenants)
	}
	if _, err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	assertVersion(t, ctx, runner, baselineVersion)
	if got := tablesAndOwners(t, ctx, db); !slices.Equal(got, afterTenants[:1]) {
		t.Errorf("tables after rolling back the tenants migration = %v, want %v", got, afterTenants[:1])
	}
	if _, err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	assertVersion(t, ctx, runner, 0)
	if _, err := runner.Down(ctx); !errors.Is(err, migration.ErrNothingToRollBack) {
		t.Errorf("Down() at version 0 = %v, want ErrNothingToRollBack", err)
	}

	// And the migrations apply cleanly again after a rollback.
	if _, err := runner.Up(ctx); err != nil {
		t.Fatalf("Up() after Down() error: %v", err)
	}
	assertVersion(t, ctx, runner, latestVersion)
	if got := tablesAndOwners(t, ctx, db); !slices.Equal(got, want) {
		t.Errorf("tables after re-applying = %v, want %v", got, want)
	}
}

// tablesAndOwners lists "schema.table owner" for every table outside the system schemas.
func tablesAndOwners(t *testing.T, ctx context.Context, db *database.DB) []string {
	t.Helper()
	rows, err := db.Pool().Query(ctx, `
		SELECT schemaname || '.' || tablename || ' ' || tableowner FROM pg_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY 1`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	return tables
}

func assertVersion(t *testing.T, ctx context.Context, r *migration.Runner, want int64) {
	t.Helper()
	got, err := r.Version(ctx)
	if err != nil {
		t.Fatalf("Version() error: %v", err)
	}
	if got != want {
		t.Errorf("Version() = %d, want %d", got, want)
	}
}

// Test-only migrations, applied to throwaway databases to exercise the runner. They are not part
// of the application schema.
func testMigrations(second string) fstest.MapFS {
	return fstest.MapFS{
		"20000101000001_first.sql": {Data: []byte(`
-- +goose Up
CREATE TABLE runner_probes (id int PRIMARY KEY);
-- +goose Down
DROP TABLE runner_probes;
`)},
		"20000101000002_second.sql": {Data: []byte(second)},
	}
}

func TestIntegrationFailedMigrationRollsBack(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	db, sdb := sqlDB(t, dbtest.NewDatabase(t))
	runner := newRunner(t, sdb, testMigrations(`
-- +goose Up
CREATE TABLE half_applied (id int);
INSERT INTO runner_probes (id) VALUES ('not a number');
-- +goose Down
DROP TABLE half_applied;
`))

	_, err := runner.Up(ctx)
	var partial *goose.PartialError
	if !errors.As(err, &partial) {
		t.Fatalf("Up() error = %v, want a *goose.PartialError", err)
	}
	if len(partial.Applied) != 1 || partial.Failed == nil {
		t.Errorf("PartialError applied %d, failed %v; want 1 applied and the second failed", len(partial.Applied), partial.Failed)
	}
	assertVersion(t, ctx, runner, 20000101000001)

	// The failing migration ran in a transaction, so its first statement was rolled back too.
	var exists bool
	if err := db.Pool().QueryRow(ctx, "SELECT to_regclass('public.half_applied') IS NOT NULL").Scan(&exists); err != nil {
		t.Fatalf("check table: %v", err)
	}
	if exists {
		t.Error("table from the failed migration exists; the migration was not atomic")
	}
}

func TestIntegrationUpWaitsForMigrationLock(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	cfg := dbtest.NewDatabase(t)

	// Another migrate process holds goose's session-level advisory lock.
	holder, _ := sqlDB(t, cfg)
	conn, err := holder.Pool().Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lock.DefaultLockID); err != nil {
		t.Fatalf("take migration lock: %v", err)
	}

	_, sdb := sqlDB(t, cfg)
	runner := newRunner(t, sdb, nil)
	done := make(chan error, 1)
	go func() {
		_, err := runner.Up(ctx)
		done <- err
	}()

	// While the lock is held, Up must wait and apply nothing.
	select {
	case err := <-done:
		t.Fatalf("Up() finished while another session held the migration lock (error: %v)", err)
	case <-time.After(1500 * time.Millisecond):
	}
	// (goose may already have created its empty version table while checking for pending
	// migrations; what matters is that no migration version has been recorded.)
	var applied int
	if err := conn.QueryRow(ctx, `
		SELECT CASE WHEN to_regclass('public.goose_db_version') IS NULL THEN 0
		ELSE (SELECT count(*) FROM goose_db_version WHERE version_id > 0) END`).Scan(&applied); err != nil {
		t.Fatalf("count applied versions: %v", err)
	}
	if applied != 0 {
		t.Errorf("%d migrations were applied while another session held the migration lock", applied)
	}

	// Once the lock is released, Up proceeds (goose retries the lock every 5 seconds).
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lock.DefaultLockID); err != nil {
		t.Fatalf("release migration lock: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Up() error after the lock was released: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Up() did not proceed after the lock was released")
	}
	assertVersion(t, ctx, runner, latestVersion)
}
