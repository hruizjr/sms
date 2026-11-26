//go:build integration

// The helpers in this file create and drop databases, so they are compiled only with the
// "integration" build tag and always go through the TargetConfig safeguards (target.go).

package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbroles"
)

const adminTimeout = 30 * time.Second

// ServerConfig returns the database settings from the process environment after the TargetConfig
// safeguards (APP_ENV=test, valid configuration, loopback DB_HOST). It fails the test with setup
// instructions otherwise; the message never contains the password.
func ServerConfig(t testing.TB) config.DatabaseConfig {
	t.Helper()
	cfg, err := TargetConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("%v\nIntegration tests need APP_ENV=test and a local PostgreSQL server configured through "+
			"the DB_* variables (see backend/README.md, \"Testing\").", err)
	}
	return cfg
}

// NewDatabase creates an empty database with a random name and returns settings that point to it,
// as the configured DB_USER (the server's bootstrap superuser). The database is dropped (terminating
// any remaining connections) during test cleanup.
//
// The application roles are provisioned in it exactly as in the local stack (dbroles.ProvisionSQL):
// sms_migrator and sms_app are created on the server if they do not exist yet (without a password)
// and receive their privileges in the new database. Use RoleConfig to connect as one of them.
func NewDatabase(t testing.TB) config.DatabaseConfig {
	t.Helper()
	server := ServerConfig(t)
	name := NamePrefix + randomSuffix(t)
	if err := checkDisposableName(name, server.Name); err != nil {
		t.Fatalf("refusing to create database: %v", err)
	}

	exec(t, server, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	t.Cleanup(func() {
		// Checked again immediately before dropping: only a generated sms_test_ name that is not the
		// configured maintenance database (DB_NAME) can ever be dropped.
		if err := checkDisposableName(name, server.Name); err != nil {
			t.Fatalf("refusing to drop database: %v", err)
		}
		exec(t, server, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
	})

	cfg := server
	cfg.Name = name
	exec(t, cfg, dbroles.ProvisionSQL)
	return cfg
}

// RoleConfig returns cfg with the user and password of an application role: dbroles.Migrator
// (password from DB_MIGRATOR_PASSWORD) or dbroles.App (password from DB_APP_PASSWORD). The roles'
// passwords are set once per server by the provisioning script (infra/local/postgres-init/
// 10-sms-roles.sh; in CI, the "Provision database roles" step); tests never change them, because
// the server may be a developer's local stack. It fails the test with setup instructions when the
// variable is not set.
func RoleConfig(t testing.TB, cfg config.DatabaseConfig, role string) config.DatabaseConfig {
	t.Helper()
	var env string
	switch role {
	case dbroles.Migrator:
		env = EnvMigratorPassword
	case dbroles.App:
		env = EnvAppPassword
	default:
		t.Fatalf("RoleConfig: unknown role %q", role)
	}
	password, ok := os.LookupEnv(env)
	if !ok || password == "" {
		t.Fatalf("%s is not set. Integration tests that connect as %s need the role's password, as set by the "+
			"role provisioning script (see backend/README.md, \"Testing\").", env, role)
	}
	cfg.User = role
	cfg.Password = config.Secret(password)
	return cfg
}

// NewName returns a database name with the test prefix that does not exist.
func NewName(t testing.TB) string {
	t.Helper()
	return NamePrefix + "missing_" + randomSuffix(t)
}

// exec runs one administrative statement on the server's maintenance database (DB_NAME).
func exec(t testing.TB, cfg config.DatabaseConfig, sql string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), adminTimeout)
	defer cancel()

	cfg.MaxConns, cfg.MinConns = 1, 0
	db, err := database.Open(ctx, cfg, "sms-integration-test")
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer db.Close()
	if _, err := db.Pool().Exec(ctx, sql); err != nil {
		t.Fatalf("%s: %v", sql, err) // the statement holds only a generated database name
	}
}

func randomSuffix(t testing.TB) string {
	t.Helper()
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random database name: %v", err)
	}
	return hex.EncodeToString(b)
}
