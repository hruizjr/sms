//go:build integration

// Package dbtest gives integration tests an isolated, throwaway PostgreSQL database.
//
// It is compiled only with the "integration" build tag. Connection settings come from the same
// DB_* environment variables as the API (for example the local Docker Compose database, or the CI
// service container). The configured DB_NAME is used only to connect to the server; tests never
// create objects in it, modify it, or drop it. Each test gets a new database named
// sms_test_<random>, which is dropped when the test ends, whether it passed or failed.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
)

// NamePrefix starts the name of every database created by this package. Only databases with this
// prefix are ever dropped.
const NamePrefix = "sms_test_"

const adminTimeout = 30 * time.Second

// ServerConfig returns the database settings from the environment. It fails the test, with
// instructions, if they are missing or invalid.
func ServerConfig(t testing.TB) config.DatabaseConfig {
	t.Helper()
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		// Configuration errors never contain the password.
		t.Fatalf("integration tests need a PostgreSQL server configured through the DB_* environment "+
			"variables (see backend/README.md, \"Integration tests\"): %v", err)
	}
	return cfg.Database
}

// NewDatabase creates an empty database with a random name and returns settings that point to it.
// The database is dropped (terminating any remaining connections) during test cleanup.
func NewDatabase(t testing.TB) config.DatabaseConfig {
	t.Helper()
	server := ServerConfig(t)
	name := NamePrefix + randomSuffix(t)

	exec(t, server, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	t.Cleanup(func() {
		if !strings.HasPrefix(name, NamePrefix) {
			t.Fatalf("refusing to drop database %q without the %q prefix", name, NamePrefix)
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
