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

// NewDatabase creates an empty database with a random name and returns settings that point to it.
// The database is dropped (terminating any remaining connections) during test cleanup.
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
