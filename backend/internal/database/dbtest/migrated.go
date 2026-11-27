//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbroles"
	"github.com/hruizjr/sms/backend/internal/migration"
	"github.com/hruizjr/sms/backend/migrations"
)

// Roles holds connection settings for one test database: as the bootstrap superuser (Admin), the
// migration role (Migrator) and the application role (App).
type Roles struct {
	Admin    config.DatabaseConfig
	Migrator config.DatabaseConfig
	App      config.DatabaseConfig
}

// NewMigratedDatabase creates a database with NewDatabase and applies every embedded migration as
// the migration role, exactly as the migrate command does, so that the tables are owned by
// sms_migrator. It needs DB_MIGRATOR_PASSWORD and DB_APP_PASSWORD (see RoleConfig).
func NewMigratedDatabase(t testing.TB) Roles {
	t.Helper()
	admin := NewDatabase(t)
	roles := Roles{
		Admin:    admin,
		Migrator: RoleConfig(t, admin, dbroles.Migrator),
		App:      RoleConfig(t, admin, dbroles.App),
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminTimeout)
	defer cancel()
	cfg := roles.Migrator
	cfg.MaxConns, cfg.MinConns = 2, 0
	db, err := database.Open(ctx, cfg, "sms-integration-test-migrate")
	if err != nil {
		t.Fatalf("open migration connection: %v", err)
	}
	defer db.Close()
	if err := database.CheckMigrationRole(ctx, db.Pool()); err != nil {
		t.Fatalf("migration role: %v", err)
	}
	sqlDB := stdlib.OpenDBFromPool(db.Pool())
	defer func() { _ = sqlDB.Close() }() // nothing to report: the pool is closed right after
	runner, err := migration.New(sqlDB, migrations.FS)
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if _, err := runner.Up(ctx); err != nil {
		t.Fatalf("apply migrations as %s: %v", dbroles.Migrator, err)
	}
	return roles
}
