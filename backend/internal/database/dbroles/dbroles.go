// Package dbroles names the PostgreSQL roles of the application and embeds the script that
// provisions them (Phase 4 Group 1, owner decision RLS-4; decision log §18).
//
//   - Migrator (sms_migrator) runs migrations and owns the application tables.
//   - App (sms_app) is the API's login role: it owns nothing, cannot create objects, and receives
//     only the table privileges that each migration grants explicitly (never TRUNCATE).
//
// Neither role is a superuser or has BYPASSRLS, CREATEROLE, CREATEDB or REPLICATION. The API checks
// its own role on every new connection (database.RequireApplicationRole) and the migrate command
// checks its role before running (database.CheckMigrationRole).
//
// ProvisionSQL is provision.sql, the single source of the role definitions. The local Docker
// Compose database runs the same file (infra/local/postgres-init/10-sms-roles.sh), and
// dbtest.NewDatabase runs it in every integration-test database. Production provisioning is out of
// scope for this group.
package dbroles

import _ "embed" // for go:embed

// Role names. Migrations grant table privileges to App by this name.
const (
	Migrator = "sms_migrator"
	App      = "sms_app"
)

// ProvisionSQL creates and enforces both roles and the privileges of the current database. It
// must be executed by a superuser connected to the application database; it is idempotent, sets no
// passwords, and never drops a role or any data. See the comment at the top of provision.sql.
//
//go:embed provision.sql
var ProvisionSQL string
