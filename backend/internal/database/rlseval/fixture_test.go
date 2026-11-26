//go:build integration && rlseval

package rlseval

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
)

// Synthetic tenant identifiers. They are not real tenants and match no real data.
const (
	tenantA = "aaaaaaaa-0000-4000-8000-00000000000a"
	tenantB = "bbbbbbbb-0000-4000-8000-00000000000b"
	tenantC = "cccccccc-0000-4000-8000-00000000000c"
	tenantD = "dddddddd-0000-4000-8000-00000000000d"
)

// Seeded rows of rlseval.items: three for tenant A, two for tenant B, none for C or D.
var seededLabels = map[string][]string{
	tenantA: {"a1", "a2", "a3"},
	tenantB: {"b1", "b2"},
}

// tenantPolicy is the fail-closed tenant expression proposed in ADR-0010: the transaction-local
// setting app.tenant_id, read with missing_ok = true. NULLIF turns the empty string that PostgreSQL
// returns once the setting has been used on a connection (see context_test.go) into NULL, so a
// missing context compares as NULL and matches no row.
const tenantPolicy = `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid`

// SQLSTATE codes asserted by the tests.
const (
	sqlStateInsufficientPrivilege = "42501" // also raised for row-level security violations
	sqlStateInvalidText           = "22P02" // invalid input syntax (e.g. for type uuid)
	sqlStateUniqueViolation       = "23505"
	sqlStateForeignKeyViolation   = "23503"
	sqlStateMissingTenant         = "P0001" // raise_exception, used by rlseval.require_tenant_id()
)

const testTimeout = 60 * time.Second

// fixture is one disposable database with its own set of NOLOGIN roles:
//
//	owner      owns the schema and tables (the analogue of the migration role)
//	app        the application runtime role: not owner, not superuser, NOBYPASSRLS
//	reporting  cross-tenant model 1: explicit, read-only policies per table
//	bypass     cross-tenant model 2: a dedicated BYPASSRLS role
//
// The test process connects as the disposable server's superuser and assumes these roles with
// SET ROLE (owner decision: test roles are NOLOGIN). Row-level security is evaluated for the
// current role, so a session that has run SET ROLE app is subject to exactly the policies that an
// application logging in as app would be. The limitation is that such a session could RESET ROLE;
// production would log in as the application role directly (rls-evaluation.md, §3).
type fixture struct {
	db                            config.DatabaseConfig
	admin                         *pgxpool.Pool // superuser, no SET ROLE: bypasses RLS
	owner, app, reporting, bypass string        // role names
}

// newFixture creates the roles, a disposable database, and the base schema:
//
//	rlseval.items  tenant-owned rows; RLS enabled and FORCED; tenant policy for app;
//	               read-only policy for reporting; seeded with seededLabels.
func newFixture(t testing.TB) *fixture {
	t.Helper()
	server := dbtest.ServerConfig(t)
	suffix := randomHex(t, 3)
	prefix := rolePrefix + runID + "_" + suffix + "_"
	f := &fixture{
		owner:     prefix + "owner",
		app:       prefix + "app",
		reporting: prefix + "reporting",
		bypass:    prefix + "bypass",
	}
	roles := []string{f.owner, f.app, f.reporting, f.bypass}

	// Registered first so that it runs last, after dbtest has dropped the database (t.Cleanup is
	// LIFO): the roles then own nothing and can be dropped.
	t.Cleanup(func() {
		for _, role := range roles {
			mustDisposableRole(t, role)
			serverExec(t, server, "DROP ROLE IF EXISTS "+ident(role))
		}
	})
	for _, role := range roles {
		mustDisposableRole(t, role)
		attrs := "NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS"
		if role == f.bypass {
			attrs = "NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT BYPASSRLS"
		}
		serverExec(t, server, "CREATE ROLE "+ident(role)+" "+attrs)
	}

	f.db = dbtest.NewDatabase(t)
	f.admin = newPool(t, f.db, "", nil)

	f.adminExec(t,
		"CREATE SCHEMA rlseval AUTHORIZATION "+ident(f.owner),
		"GRANT USAGE ON SCHEMA rlseval TO "+ident(f.app)+", "+ident(f.reporting)+", "+ident(f.bypass),
	)
	f.ownerExec(t,
		`CREATE TABLE rlseval.items (
			tenant_id uuid NOT NULL,
			id        uuid NOT NULL DEFAULT gen_random_uuid(),
			label     text NOT NULL,
			CONSTRAINT pk_items PRIMARY KEY (tenant_id, id)
		)`,
		`ALTER TABLE rlseval.items ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE rlseval.items FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY items_tenant_isolation ON rlseval.items FOR ALL TO `+ident(f.app)+
			` USING (`+tenantPolicy+`) WITH CHECK (`+tenantPolicy+`)`,
		`CREATE POLICY items_reporting_read ON rlseval.items FOR SELECT TO `+ident(f.reporting)+` USING (true)`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON rlseval.items TO `+ident(f.app),
		// INSERT is granted on purpose: the reporting role still cannot insert, because no policy
		// allows it (default deny), which is what crosstenant_test.go shows.
		`GRANT SELECT, INSERT ON rlseval.items TO `+ident(f.reporting),
		`GRANT SELECT ON rlseval.items TO `+ident(f.bypass),
	)
	for tenant, labels := range seededLabels {
		for _, label := range labels {
			f.adminExec(t, fmt.Sprintf(`INSERT INTO rlseval.items (tenant_id, label) VALUES ('%s', '%s')`, tenant, label))
		}
	}
	return f
}

// pool returns a pool whose connections act as role (SET ROLE right after connecting). maxConns
// bounds the pool; tune, if not nil, adjusts the configuration further.
func (f *fixture) pool(t testing.TB, role string, maxConns int32, tune func(*pgxpool.Config)) *pgxpool.Pool {
	t.Helper()
	return newPool(t, f.db, role, func(c *pgxpool.Config) {
		c.MaxConns = maxConns
		if tune != nil {
			tune(c)
		}
	})
}

// adminExec runs statements as the superuser (which bypasses row-level security).
func (f *fixture) adminExec(t testing.TB, statements ...string) {
	t.Helper()
	ctx := testContext(t)
	for _, s := range statements {
		if _, err := f.admin.Exec(ctx, s); err != nil {
			t.Fatalf("admin: %s: %v", firstLine(s), err)
		}
	}
}

// ownerExec runs statements as the owner role, in one transaction, so that created objects are
// owned by it.
func (f *fixture) ownerExec(t testing.TB, statements ...string) {
	t.Helper()
	ctx := testContext(t)
	err := pgx.BeginFunc(ctx, f.admin, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+ident(f.owner)); err != nil {
			return err
		}
		for _, s := range statements {
			if _, err := tx.Exec(ctx, s); err != nil {
				return fmt.Errorf("%s: %w", firstLine(s), err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("owner: %v", err)
	}
}

// newPool opens a pool on cfg's database. With a non-empty role, every new connection runs
// SET ROLE role before it is used; pgxpool does not reset session state between acquisitions, so
// the role stays in effect for the life of the connection.
func newPool(t testing.TB, cfg config.DatabaseConfig, role string, tune func(*pgxpool.Config)) *pgxpool.Pool {
	t.Helper()
	pc, err := pgxpool.ParseConfig(connString(cfg))
	if err != nil {
		t.Fatalf("parse pool config: %v", errors.New("invalid connection settings")) // never echo the string
	}
	pc.MaxConns = 4
	pc.MinConns = 0
	if role != "" {
		setRole := "SET ROLE " + ident(role)
		pc.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, setRole)
			return err
		}
	}
	if tune != nil {
		tune(pc)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), pc)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// connString builds a keyword/value connection string, quoting each value as libpq does (the same
// approach as internal/database). It holds the password and is never logged.
func connString(cfg config.DatabaseConfig) string {
	kv := []struct{ k, v string }{
		{"host", cfg.Host},
		{"port", strconv.Itoa(cfg.Port)},
		{"dbname", cfg.Name},
		{"user", cfg.User},
		{"password", cfg.Password.Reveal()},
		{"sslmode", string(cfg.SSLMode)},
		{"application_name", "sms-rlseval"},
	}
	var b strings.Builder
	for _, p := range kv {
		v := strings.ReplaceAll(strings.ReplaceAll(p.v, `\`, `\\`), `'`, `\'`)
		b.WriteString(p.k + "='" + v + "' ")
	}
	return b.String()
}

// serverExec runs one statement on the server's maintenance database (role DDL is server-wide).
func serverExec(t testing.TB, server config.DatabaseConfig, statement string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, connString(server))
	if err != nil {
		t.Fatalf("connect to maintenance database: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }() //nolint:errcheck // closing a read-only admin connection; nothing to recover
	if _, err := conn.Exec(ctx, statement); err != nil {
		t.Fatalf("%s: %v", statement, err)
	}
}

// mustDisposableRole refuses any role name that these tests did not generate, so that no existing
// application role can ever be created over, altered, or dropped.
func mustDisposableRole(t testing.TB, role string) {
	t.Helper()
	want := rolePrefix + runID + "_"
	if !strings.HasPrefix(role, want) || len(role) > 63 {
		t.Fatalf("refusing role %q: not a generated %s<suffix> name", role, want)
	}
	for _, c := range role {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			t.Fatalf("refusing role %q: character outside [a-z0-9_]", role)
		}
	}
}

// inTenant runs fn in a transaction on pool after setting app.tenant_id transaction-locally.
// An empty tenant runs fn with no tenant context at all.
func inTenant(ctx context.Context, pool *pgxpool.Pool, tenant string, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if tenant != "" {
			if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenant); err != nil {
				return err
			}
		}
		return fn(tx)
	})
}

// visible returns the (tenant_id, label) pairs of rlseval.items visible in tx, sorted.
func visible(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, table string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT tenant_id::text || '/' || label FROM `+table+` ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// rowsOf returns the "tenant/label" strings that seededLabels holds for the given tenants, sorted.
func rowsOf(tenants ...string) []string {
	var out []string
	for _, tenant := range []string{tenantA, tenantB} { // sorted order of the seeded tenants
		for _, want := range tenants {
			if want == tenant {
				for _, l := range seededLabels[tenant] {
					out = append(out, tenant+"/"+l)
				}
			}
		}
	}
	return out
}

func equalRows(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// wantSQLState fails the test unless err is a PostgreSQL error with the given SQLSTATE.
func wantSQLState(t testing.TB, err error, code string) {
	t.Helper()
	if pgErr := asPgError(t, err); pgErr.Code != code {
		t.Fatalf("SQLSTATE = %s (%s), want %s", pgErr.Code, pgErr.Message, code)
	}
}

// asPgError fails the test unless err is a PostgreSQL error, and returns it.
func asPgError(t testing.TB, err error) *pgconn.PgError {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error = %v, want a PostgreSQL error", err)
	}
	return pgErr
}

func testContext(t testing.TB) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return ctx
}

func ident(name string) string { return pgx.Identifier{name}.Sanitize() }

func randomHex(t testing.TB, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(b)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " ..."
	}
	return s
}
