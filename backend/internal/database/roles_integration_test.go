//go:build integration

package database_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbroles"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/migration"
	"github.com/hruizjr/sms/backend/migrations"
)

// Database role separation (Phase 4 Group 1, RLS-4; decision log §18): the roles' attributes and
// privileges, the checks of the API and migration roles, and role provisioning on fresh and
// existing databases.

// mustExec runs statements on db, failing the test on the first error.
func mustExec(t *testing.T, ctx context.Context, db *database.DB, statements ...string) {
	t.Helper()
	for _, sql := range statements {
		if _, err := db.Pool().Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
}

func value[T any](t *testing.T, ctx context.Context, db *database.DB, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := db.Pool().QueryRow(ctx, sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func TestIntegrationProvisionedRoleAttributes(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	roles := dbtest.NewMigratedDatabase(t)
	admin := open(t, roles.Admin)

	for _, role := range []string{dbroles.Migrator, dbroles.App} {
		got := value[string](t, ctx, admin, `
			SELECT format('super=%s bypassrls=%s createrole=%s createdb=%s replication=%s inherit=%s login=%s',
			              rolsuper, rolbypassrls, rolcreaterole, rolcreatedb, rolreplication, rolinherit, rolcanlogin)
			FROM pg_roles WHERE rolname = $1`, role)
		want := "super=f bypassrls=f createrole=f createdb=f replication=f inherit=f login=t"
		if got != want {
			t.Errorf("%s attributes: %s, want %s", role, got, want)
		}
		if n := value[int](t, ctx, admin, `
			SELECT count(*) FROM pg_auth_members WHERE member = $1::regrole OR roleid = $1::regrole`, role); n != 0 {
			t.Errorf("%s has %d role memberships, want none", role, n)
		}
	}
	// Ownership: the migration role owns the application tables; the application role owns nothing.
	owners := value[[]string](t, ctx, admin, `
		SELECT array_agg(tablename || ' ' || tableowner ORDER BY tablename) FROM pg_tables WHERE schemaname = 'public'`)
	if want := []string{
		"api_keys sms_migrator", "dispatch_jobs sms_migrator", "goose_db_version sms_migrator",
		"idempotency_keys sms_migrator", "message_bodies sms_migrator", "message_receipts sms_migrator",
		"message_status_history sms_migrator", "messages sms_migrator", "provider_receipts sms_migrator",
		"sender_ids sms_migrator", "submission_attempts sms_migrator", "tenants sms_migrator",
	}; !slices.Equal(owners, want) {
		t.Errorf("table owners = %v, want %v", owners, want)
	}
	if n := value[int](t, ctx, admin, `SELECT count(*) FROM pg_class WHERE relowner = 'sms_app'::regrole`); n != 0 {
		t.Errorf("sms_app owns %d relations, want 0", n)
	}
	// The application role's table privileges are exactly those the tenancy, API key and messaging
	// code needs: no DELETE or TRUNCATE anywhere; history and receipt tables are insert-only; columns
	// the database sets (IDs, statuses on insert, timestamps) cannot be inserted.
	privileges := value[[]string](t, ctx, admin, `
		SELECT array_agg(p ORDER BY p) FROM (
		    SELECT table_name || ' ' || privilege_type AS p FROM information_schema.role_table_grants
		     WHERE grantee = 'sms_app'
		    UNION ALL
		    SELECT table_name || '.' || column_name || ' ' || privilege_type FROM information_schema.column_privileges
		     WHERE grantee = 'sms_app' AND privilege_type <> 'SELECT') x`)
	var want []string
	grant := func(table string, tablePrivileges []string, columns map[string][]string) {
		for _, p := range tablePrivileges {
			want = append(want, table+" "+p)
		}
		for privilege, cols := range columns {
			for _, c := range cols {
				want = append(want, table+"."+c+" "+privilege)
			}
		}
	}
	grant("tenants", []string{"SELECT"}, map[string][]string{"INSERT": {"display_name"}, "UPDATE": {"status", "updated_at"}})
	grant("api_keys", []string{"SELECT"}, map[string][]string{
		"INSERT": {"tenant_id", "label", "secret_hash", "scopes"}, "UPDATE": {"revoked_at"}})
	grant("sender_ids", []string{"SELECT", "INSERT"}, map[string][]string{"INSERT": {"tenant_id", "sender_id", "approved_at"}})
	grant("messages", []string{"SELECT"}, map[string][]string{
		"INSERT": {"tenant_id", "client_reference", "destination", "sender_id", "encoding", "segment_count"},
		"UPDATE": {"status", "status_reason", "status_updated_at", "provider_id", "has_conflicting_reports", "updated_at"}})
	grant("message_bodies", []string{"SELECT", "INSERT"}, map[string][]string{"INSERT": {"tenant_id", "message_id", "body"}})
	grant("idempotency_keys", []string{"SELECT"}, map[string][]string{
		"INSERT": {"tenant_id", "operation", "idempotency_key", "request_fingerprint", "message_id", "expires_at"},
		"UPDATE": {"request_fingerprint", "message_id", "created_at", "expires_at"}})
	grant("dispatch_jobs", []string{"SELECT"}, map[string][]string{
		"INSERT": {"tenant_id", "message_id"}, "UPDATE": {"available_at", "lease_expires_at", "completed_at", "updated_at"}})
	grant("submission_attempts", []string{"SELECT"}, map[string][]string{
		"INSERT": {"tenant_id", "message_id", "attempt_number", "provider_id"},
		"UPDATE": {"state", "provider_message_ref", "provider_result_code", "finished_at"}})
	grant("provider_receipts", []string{"SELECT"}, map[string][]string{
		"INSERT": {"provider_id", "provider_message_ref", "identity_key", "receipt_status", "provider_status_code", "provider_reported_at", "raw"},
		"UPDATE": {"matched_at"}})
	grant("message_status_history", []string{"SELECT", "INSERT"}, map[string][]string{"INSERT": {
		"id", "tenant_id", "message_id", "from_status", "to_status", "status_reason", "source", "attempt_id", "receipt_id", "occurred_at"}})
	grant("message_receipts", []string{"SELECT", "INSERT"}, map[string][]string{"INSERT": {
		"id", "tenant_id", "message_id", "provider_receipt_id", "identity_key", "receipt_status", "effect", "processed_at"}})
	slices.Sort(want)
	if !slices.Equal(privileges, want) {
		t.Errorf("sms_app privileges = %v, want %v", privileges, want)
	}
}

// TestIntegrationApplicationRoleCannotCreateOrBypass connects as sms_app and tries everything the
// role must not be able to do.
func TestIntegrationApplicationRoleCannotCreateOrBypass(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	roles := dbtest.NewMigratedDatabase(t)
	app := open(t, roles.App)

	if who := value[string](t, ctx, app, `SELECT current_user || ' ' || session_user`); who != "sms_app sms_app" {
		t.Errorf("connected as %q, want sms_app", who)
	}
	denied := []string{
		`CREATE TABLE public.app_made (id int)`,
		`CREATE TABLE app_made_unqualified (id int)`,
		`CREATE VIEW app_view AS SELECT 1`,
		`CREATE FUNCTION app_fn() RETURNS int LANGUAGE sql AS 'SELECT 1'`,
		`CREATE SCHEMA app_schema`,
		`CREATE TEMPORARY TABLE app_temp (id int)`,
		`TRUNCATE tenants`,
		`DELETE FROM tenants`,
		`INSERT INTO tenants (id, display_name) VALUES (uuidv7(), 'Chosen ID')`,
		`INSERT INTO tenants (display_name, status) VALUES ('Chosen status', 'active')`,
		`UPDATE tenants SET display_name = 'x'`,
		`ALTER TABLE tenants DISABLE ROW LEVEL SECURITY`,
		`DROP TABLE tenants`,
		`SELECT count(*) FROM goose_db_version`,
		`SET ROLE sms_migrator`,
		// Messaging tables (Phase 4 Group 2): no deletion, no rewriting of history or receipts, no
		// moving a row to another tenant, no choosing database-set columns.
		`TRUNCATE messages`,
		`DELETE FROM messages`,
		`DELETE FROM api_keys`,
		`UPDATE messages SET tenant_id = tenant_id`,
		`UPDATE messages SET destination = destination`,
		`UPDATE api_keys SET secret_hash = secret_hash`,
		`UPDATE message_status_history SET source = source`,
		`DELETE FROM message_status_history`,
		`UPDATE message_receipts SET effect = effect`,
		`DELETE FROM provider_receipts`,
		`UPDATE provider_receipts SET receipt_status = receipt_status`,
		`INSERT INTO messages (tenant_id, destination, sender_id, encoding, segment_count, status) VALUES (uuidv7(), '+639000000001', 'X', 'gsm7', 1, 'delivered')`,
		`ALTER TABLE messages DISABLE ROW LEVEL SECURITY`,
	}
	for _, sql := range denied {
		_, err := app.Pool().Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || (pgErr.Code != "42501" && pgErr.Code != "42939") {
			t.Errorf("as sms_app, %q: error = %v, want permission denied (42501)", sql, err)
		}
	}
}

func TestIntegrationRequireApplicationRole(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	roles := dbtest.NewMigratedDatabase(t)

	// The application role passes, on every connection of the pool.
	cfg := roles.App
	cfg.MinConns = 2
	app, err := database.Open(ctx, cfg, "sms-roleguard-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	if err := app.Ping(ctx); err != nil {
		t.Fatalf("Ping() as sms_app with RequireApplicationRole: %v", err)
	}

	// Every other role is refused, with the reasons from the catalog (lists in the database's
	// collation order, in which "messages" sorts before "message_status_history").
	refused := []struct {
		name string
		cfg  config.DatabaseConfig
		want []string
	}{
		{"bootstrap superuser", roles.Admin, []string{"is a superuser", "has BYPASSRLS", "has CREATEROLE"}},
		{"migration role", roles.Migrator, []string{
			"owns tables or other relations: api_keys, dispatch_jobs, goose_db_version, goose_db_version_id_seq, " +
				"idempotency_keys, message_bodies, message_receipts, messages, message_status_history, provider_receipts, " +
				"sender_ids, submission_attempts, tenants",
			"may create objects in schemas: public",
			"has TRUNCATE on: api_keys, dispatch_jobs, goose_db_version, idempotency_keys, message_bodies, message_receipts, " +
				"messages, message_status_history, provider_receipts, sender_ids, submission_attempts, tenants"}},
	}
	for _, tc := range refused {
		db, err := database.Open(ctx, tc.cfg, "sms-roleguard-test", database.RequireApplicationRole())
		if err != nil {
			t.Fatal(err)
		}
		err = db.Ping(ctx)
		db.Close()
		var unsafe *database.UnsafeRoleError
		if !errors.Is(err, database.ErrUnsafeRole) || !errors.As(err, &unsafe) {
			t.Errorf("%s: Ping() = %v, want an UnsafeRoleError", tc.name, err)
			continue
		}
		for _, want := range tc.want {
			if !slices.Contains(unsafe.Problems, want) {
				t.Errorf("%s: problems %q do not include %q", tc.name, unsafe.Problems, want)
			}
		}
		if strings.Contains(err.Error(), tc.cfg.Password.Reveal()) {
			t.Errorf("%s: the error contains the password", tc.name)
		}
	}
}

// TestIntegrationApplicationRoleCheckDetectsExtraPrivileges grants the application role, in one
// database only, each privilege it must not have, and checks that the guard reports it.
func TestIntegrationApplicationRoleCheckDetectsExtraPrivileges(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	roles := dbtest.NewMigratedDatabase(t)
	admin := open(t, roles.Admin)
	app := open(t, roles.App)

	if err := database.CheckApplicationRole(ctx, app.Pool()); err != nil {
		t.Fatalf("CheckApplicationRole() before any change: %v", err)
	}
	cases := []struct {
		grant, revoke, want string
	}{
		{`GRANT CREATE ON SCHEMA public TO sms_app`, `REVOKE CREATE ON SCHEMA public FROM sms_app`,
			"may create objects in schemas: public"},
		{`GRANT TRUNCATE ON tenants TO sms_app`, `REVOKE TRUNCATE ON tenants FROM sms_app`,
			"has TRUNCATE on: tenants"},
		{`GRANT CREATE ON DATABASE ` + pgx.Identifier{roles.Admin.Name}.Sanitize() + ` TO sms_app`,
			`REVOKE CREATE ON DATABASE ` + pgx.Identifier{roles.Admin.Name}.Sanitize() + ` FROM sms_app`,
			"may create schemas in the database"},
		{`CREATE TABLE public.handed_over (id int); ALTER TABLE public.handed_over OWNER TO sms_app`,
			`DROP TABLE public.handed_over`, "owns tables or other relations: handed_over"},
		{`CREATE SCHEMA app_owned AUTHORIZATION sms_app`, `DROP SCHEMA app_owned`, "owns schemas: app_owned"},
	}
	for _, tc := range cases {
		mustExec(t, ctx, admin, tc.grant)
		err := database.CheckApplicationRole(ctx, app.Pool())
		var unsafe *database.UnsafeRoleError
		if !errors.As(err, &unsafe) || !slices.Contains(unsafe.Problems, tc.want) {
			t.Errorf("after %q: CheckApplicationRole() = %v, want a problem %q", tc.grant, err, tc.want)
		}
		mustExec(t, ctx, admin, tc.revoke)
		if err := database.CheckApplicationRole(ctx, app.Pool()); err != nil {
			t.Errorf("after %q: CheckApplicationRole() = %v, want nil", tc.revoke, err)
		}
	}
}

func TestIntegrationCheckMigrationRole(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	roles := dbtest.NewMigratedDatabase(t)
	if err := database.CheckMigrationRole(ctx, open(t, roles.Migrator).Pool()); err != nil {
		t.Errorf("CheckMigrationRole() as sms_migrator = %v, want nil", err)
	}
	err := database.CheckMigrationRole(ctx, open(t, roles.Admin).Pool())
	var unsafe *database.UnsafeRoleError
	if !errors.As(err, &unsafe) || !slices.Contains(unsafe.Problems, "is a superuser") {
		t.Errorf("CheckMigrationRole() as the superuser = %v, want a refusal", err)
	}
}

// aclSnapshot returns the privileges and owners provisioning manages, to compare runs.
func aclSnapshot(t *testing.T, ctx context.Context, admin *database.DB) string {
	t.Helper()
	return value[string](t, ctx, admin, `
		SELECT concat_ws(' | ',
		    (SELECT datacl::text FROM pg_database WHERE datname = current_database()),
		    (SELECT nspacl::text FROM pg_namespace WHERE nspname = 'public'),
		    (SELECT string_agg(defaclacl::text, ',') FROM pg_default_acl),
		    (SELECT string_agg(relname || ':' || pg_get_userbyid(relowner) || ':' || coalesce(relacl::text, ''), ','
		                       ORDER BY relname) FROM pg_class WHERE relnamespace = 'public'::regnamespace))`)
}

func TestIntegrationProvisioningIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	roles := dbtest.NewMigratedDatabase(t) // provisioned once, then migrated
	admin := open(t, roles.Admin)
	before := aclSnapshot(t, ctx, admin)
	for range 2 {
		mustExec(t, ctx, admin, dbroles.ProvisionSQL)
	}
	if after := aclSnapshot(t, ctx, admin); after != before {
		t.Errorf("provisioning again changed privileges or owners:\nbefore %s\nafter  %s", before, after)
	}
	if n := value[int](t, ctx, admin, `SELECT count(*) FROM tenants`); n != 0 {
		t.Errorf("tenants has %d rows", n)
	}
}

// TestIntegrationProvisioningUpgradesAnExistingDatabase reproduces a database created before role
// separation: the baseline migration applied by the bootstrap superuser, so goose's version table
// belongs to it. Provisioning transfers that table to sms_migrator and keeps its rows; the migration
// role can then apply the remaining migrations.
func TestIntegrationProvisioningUpgradesAnExistingDatabase(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	adminCfg := dbtest.NewDatabase(t)
	admin := open(t, adminCfg)

	// The pre-role-separation state: privileges of a plain database, baseline applied as superuser.
	// (Not DROP OWNED BY: it would also revoke the roles' privileges on every other database.)
	dbName := pgx.Identifier{adminCfg.Name}.Sanitize()
	mustExec(t, ctx, admin,
		`REVOKE ALL ON DATABASE `+dbName+` FROM sms_migrator, sms_app`,
		`REVOKE ALL ON SCHEMA public FROM sms_migrator, sms_app`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE sms_migrator GRANT EXECUTE ON FUNCTIONS TO PUBLIC`,
		`GRANT CONNECT, TEMPORARY ON DATABASE `+dbName+` TO PUBLIC`,
		`GRANT USAGE ON SCHEMA public TO PUBLIC`)
	baseline, err := migrations.FS.ReadFile("20261125041208_baseline.sql")
	if err != nil {
		t.Fatal(err)
	}
	runMigrations(t, ctx, admin, fstest.MapFS{"20261125041208_baseline.sql": {Data: baseline}})
	if owner := value[string](t, ctx, admin, `SELECT tableowner FROM pg_tables WHERE tablename = 'goose_db_version'`); owner != adminCfg.User {
		t.Fatalf("goose_db_version owner = %s, want the bootstrap role %s", owner, adminCfg.User)
	}
	rowsBefore := value[string](t, ctx, admin, `SELECT string_agg(version_id || ':' || tstamp, ',' ORDER BY id) FROM goose_db_version`)

	// The upgrade: provisioning, then the migrate command's work as the migration role.
	mustExec(t, ctx, admin, dbroles.ProvisionSQL)
	if owner := value[string](t, ctx, admin, `SELECT tableowner FROM pg_tables WHERE tablename = 'goose_db_version'`); owner != dbroles.Migrator {
		t.Errorf("after provisioning, goose_db_version owner = %s, want %s", owner, dbroles.Migrator)
	}
	if rowsAfter := value[string](t, ctx, admin, `SELECT string_agg(version_id || ':' || tstamp, ',' ORDER BY id) FROM goose_db_version`); rowsAfter != rowsBefore {
		t.Errorf("goose_db_version rows changed: %s -> %s", rowsBefore, rowsAfter)
	}
	migrator := open(t, dbtest.RoleConfig(t, adminCfg, dbroles.Migrator))
	if err := database.CheckMigrationRole(ctx, migrator.Pool()); err != nil {
		t.Fatal(err)
	}
	runMigrations(t, ctx, migrator, migrations.FS)
	if v := value[int64](t, ctx, admin, `SELECT max(version_id) FROM goose_db_version WHERE is_applied`); v != 20261127150000 {
		t.Errorf("version after upgrade = %d", v)
	}
	if err := database.CheckApplicationRole(ctx, open(t, dbtest.RoleConfig(t, adminCfg, dbroles.App)).Pool()); err != nil {
		t.Errorf("application role after upgrade: %v", err)
	}
}

func TestIntegrationProvisioningRefusesUnsafeStates(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	adminCfg := dbtest.NewDatabase(t)
	admin := open(t, adminCfg)

	// An object in schema public owned by someone else is reported, not taken over or dropped.
	mustExec(t, ctx, admin, `CREATE TABLE public.left_behind (id int)`, `INSERT INTO public.left_behind VALUES (7)`)
	before := aclSnapshot(t, ctx, admin)
	_, err := admin.Pool().Exec(ctx, dbroles.ProvisionSQL)
	if err == nil || !strings.Contains(err.Error(), "left_behind (owner "+adminCfg.User+")") {
		t.Errorf("provisioning with a foreign-owned table: error = %v, want it to name left_behind", err)
	}
	if after := aclSnapshot(t, ctx, admin); after != before {
		t.Errorf("a failed provisioning changed privileges or owners:\nbefore %s\nafter  %s", before, after)
	}
	if n := value[int](t, ctx, admin, `SELECT id FROM public.left_behind`); n != 7 {
		t.Errorf("left_behind data changed: %d", n)
	}
	mustExec(t, ctx, admin, `DROP TABLE public.left_behind`)

	// Only a superuser may provision.
	migrator := open(t, dbtest.RoleConfig(t, adminCfg, dbroles.Migrator))
	if _, err := migrator.Pool().Exec(ctx, dbroles.ProvisionSQL); err == nil ||
		!strings.Contains(err.Error(), "must run as a superuser") {
		t.Errorf("provisioning as sms_migrator: error = %v, want a refusal", err)
	}
}

func runMigrations(t *testing.T, ctx context.Context, db *database.DB, fsys fs.FS) {
	t.Helper()
	sqlDB := stdlib.OpenDBFromPool(db.Pool())
	defer func() { _ = sqlDB.Close() }() // the pool stays open; nothing to report
	runner, err := migration.New(sqlDB, fsys)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Up(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}
