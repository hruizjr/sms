package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Checks of the database role a process is connected as (Phase 4 Group 1, owner decision RLS-4;
// decision log §18). Row-level security protects tenant data only if the API's role cannot bypass
// it: superusers, roles with BYPASSRLS and table owners are not subject to (non-forced) policies,
// and a role that can create objects or truncate tables can work around them. The checks read the
// PostgreSQL catalog of the connected database; they change nothing.

// ErrUnsafeRole is wrapped by every UnsafeRoleError.
var ErrUnsafeRole = errors.New("unsafe database role")

// UnsafeRoleError lists why the connected role must not be used for its purpose.
type UnsafeRoleError struct {
	Purpose  string   // "application" or "migration"
	Role     string   // the connected role (current_user)
	Problems []string // one entry per finding, from the catalog
}

func (e *UnsafeRoleError) Error() string {
	return fmt.Sprintf("refusing to use database role %q as the %s role: %s",
		e.Role, e.Purpose, strings.Join(e.Problems, "; "))
}

// Unwrap makes errors.Is(err, ErrUnsafeRole) true.
func (e *UnsafeRoleError) Unwrap() error { return ErrUnsafeRole }

// Querier is satisfied by *pgx.Conn, *pgxpool.Pool, *pgxpool.Conn and pgx.Tx.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// privilegedRoleChecks are the findings common to both purposes: role attributes, and membership
// in any other role (through which privileges could be inherited or assumed with SET ROLE). r is
// the connected role's pg_roles row.
const privilegedRoleChecks = `
	CASE WHEN r.rolsuper THEN 'is a superuser' END,
	CASE WHEN r.rolbypassrls THEN 'has BYPASSRLS' END,
	CASE WHEN r.rolcreaterole THEN 'has CREATEROLE' END,
	CASE WHEN r.rolcreatedb THEN 'has CREATEDB' END,
	CASE WHEN r.rolreplication THEN 'has REPLICATION' END,
	(SELECT 'is a member of ' || string_agg(m.rolname, ', ' ORDER BY m.rolname)
	   FROM pg_roles m
	  WHERE m.oid <> r.oid AND pg_has_role(r.oid, m.oid, 'MEMBER'))`

// applicationRoleQuery returns the connected role and its findings as the application role: the
// common checks, plus ownership of anything in this database, permission to create objects, and
// TRUNCATE on any table. Objects are listed from the application schemas only (every schema except
// PostgreSQL's own), so that the findings stay short; ownership of system objects implies being a
// superuser, which is reported above.
const applicationRoleQuery = `
WITH app_schemas AS (
	SELECT oid, nspname FROM pg_namespace
	WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
	  AND nspname NOT LIKE 'pg\_temp\_%' AND nspname NOT LIKE 'pg\_toast\_temp\_%'
)
SELECT r.rolname::text, array_remove(ARRAY[` + privilegedRoleChecks + `,
	CASE WHEN (SELECT datdba FROM pg_database WHERE datname = current_database()) = r.oid
	     THEN 'owns the database' END,
	CASE WHEN has_database_privilege(r.oid, current_database(), 'CREATE')
	     THEN 'may create schemas in the database' END,
	(SELECT 'owns schemas: ' || string_agg(n.nspname, ', ' ORDER BY n.nspname)
	   FROM pg_namespace n WHERE n.nspowner = r.oid
	    AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')),
	(SELECT 'owns tables or other relations: ' || string_agg(c.oid::regclass::text, ', ' ORDER BY c.oid::regclass::text)
	   FROM pg_class c JOIN app_schemas n ON n.oid = c.relnamespace
	  WHERE c.relowner = r.oid AND c.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')),
	(SELECT 'owns functions: ' || string_agg(p.oid::regprocedure::text, ', ' ORDER BY p.oid::regprocedure::text)
	   FROM pg_proc p JOIN app_schemas n ON n.oid = p.pronamespace
	  WHERE p.proowner = r.oid),
	(SELECT 'may create objects in schemas: ' || string_agg(n.nspname, ', ' ORDER BY n.nspname)
	   FROM app_schemas n WHERE has_schema_privilege(r.oid, n.oid, 'CREATE')),
	(SELECT 'has TRUNCATE on: ' || string_agg(c.oid::regclass::text, ', ' ORDER BY c.oid::regclass::text)
	   FROM pg_class c JOIN app_schemas n ON n.oid = c.relnamespace
	  WHERE c.relkind IN ('r', 'p') AND has_table_privilege(r.oid, c.oid, 'TRUNCATE'))
], NULL)
FROM pg_roles r
WHERE r.rolname = current_user`

// migrationRoleQuery returns the connected role and its findings as the migration role. The
// migration role owns the application tables and may create objects, so only the common checks
// apply.
const migrationRoleQuery = `
SELECT r.rolname::text, array_remove(ARRAY[` + privilegedRoleChecks + `
], NULL)
FROM pg_roles r
WHERE r.rolname = current_user`

// CheckApplicationRole returns an *UnsafeRoleError if the connected role must not be used by the
// API: it is a superuser or has BYPASSRLS, CREATEROLE, CREATEDB or REPLICATION; it is a member of
// any other role; it owns the database, a schema, a table or other relation, or a function; it may
// create schemas or create objects in any schema; or it has TRUNCATE on any table.
func CheckApplicationRole(ctx context.Context, q Querier) error {
	return checkRole(ctx, q, "application", applicationRoleQuery)
}

// CheckMigrationRole returns an *UnsafeRoleError if the connected role must not run migrations: it
// is a superuser or has BYPASSRLS, CREATEROLE, CREATEDB or REPLICATION, or it is a member of any
// other role. Migrations must run as the dedicated, unprivileged owner role (sms_migrator), so that
// tables are owned by it and never by the bootstrap superuser.
func CheckMigrationRole(ctx context.Context, q Querier) error {
	return checkRole(ctx, q, "migration", migrationRoleQuery)
}

func checkRole(ctx context.Context, q Querier, purpose, query string) error {
	var role string
	var problems []string
	if err := q.QueryRow(ctx, query).Scan(&role, &problems); err != nil {
		return fmt.Errorf("check %s database role: %w", purpose, err)
	}
	if len(problems) > 0 {
		return &UnsafeRoleError{Purpose: purpose, Role: role, Problems: problems}
	}
	return nil
}

// RequireApplicationRole makes the pool run CheckApplicationRole on every new connection, before
// the connection is used. A connection whose role is unsafe is closed, and acquiring it (and
// therefore Ping, readiness and every query) fails with an error wrapping ErrUnsafeRole. The API
// uses it so that it can never serve traffic through an unsafe role, even if the role's
// privileges change while the API is running (new connections are checked).
func RequireApplicationRole() Option {
	return func(c *pgxpool.Config) {
		c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			return CheckApplicationRole(ctx, conn)
		}
	}
}
