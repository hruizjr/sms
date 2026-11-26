//go:build integration && rlseval

// Package rlseval is the PostgreSQL Row-Level Security evaluation spike (Phase 3 Group 7, ADR-0010,
// D-141). It contains only tests and benchmarks: it is evidence for the owner's adoption decision,
// not product code. The report is docs/architecture/rls-evaluation.md.
//
// Everything here works on synthetic, test-only tables (rlseval.*) inside disposable databases.
// Nothing here is a migration, a business table, or a pattern that product code may import.
//
// Safety. PostgreSQL roles belong to the whole server (cluster), not to one database, so a
// disposable database alone does not isolate these tests. They therefore run only:
//
//   - with both build tags "integration" and "rlseval" (the ordinary integration run excludes them);
//   - with RLS_EVAL_DISPOSABLE_SERVER=1, which states that the server is a throwaway instance (the
//     CI service container, or the container started by scripts/rls-evaluation.sh);
//   - with the dbtest safeguards (APP_ENV=test, loopback DB_HOST);
//   - against a server whose maintenance database (DB_NAME) has never been migrated, which
//     refuses the persistent database of the local Compose stack.
//
// Every role they create is NOLOGIN and named rlseval_<run>_<fixture>_<purpose>. Roles are dropped
// in test cleanup, and TestMain fails the run if any rlseval_ role is left on the server.
package rlseval

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
)

const (
	// envDisposableServer must be "1" to confirm that DB_HOST/DB_PORT point at a throwaway server.
	envDisposableServer = "RLS_EVAL_DISPOSABLE_SERVER"
	// rolePrefix starts the name of every role created by these tests, and of no other role.
	rolePrefix = "rlseval_"
)

// runID distinguishes the roles of one test run from those of any other run.
var runID string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintf(os.Stderr, "rlseval: random run ID: %v\n", err)
		return 1
	}
	runID = hex.EncodeToString(b)

	server, err := checkServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rlseval: %v\n", err)
		return 1
	}

	code := m.Run()

	// Leftover check: no test-created role may remain on the server, from this run or any other.
	leftover, err := leftoverRoles(server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rlseval: checking for leftover roles: %v\n", err)
		return 1
	}
	if len(leftover) > 0 {
		fmt.Fprintf(os.Stderr, "rlseval: FAIL: test-created roles remain on the server: %v\n", leftover)
		return 1
	}
	fmt.Printf("rlseval: no %s* roles remain on the server\n", rolePrefix)
	return code
}

// checkServer applies the safeguards described in the package comment and returns the settings of
// the server's maintenance database.
func checkServer() (config.DatabaseConfig, error) {
	if os.Getenv(envDisposableServer) != "1" {
		return config.DatabaseConfig{}, fmt.Errorf("%s=1 is not set. These tests create server-wide roles and must "+
			"run against a disposable PostgreSQL instance: use scripts/rls-evaluation.sh, never the local "+
			"Compose stack's database", envDisposableServer)
	}
	server, err := dbtest.TargetConfig(os.LookupEnv)
	if err != nil {
		return config.DatabaseConfig{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.MaxConns, server.MinConns = 1, 0
	db, err := database.Open(ctx, server, "sms-rlseval")
	if err != nil {
		return config.DatabaseConfig{}, err
	}
	defer db.Close()

	var migrated bool
	if err := db.Pool().QueryRow(ctx,
		`SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&migrated); err != nil {
		return config.DatabaseConfig{}, fmt.Errorf("inspect maintenance database: %w", err)
	}
	if migrated {
		return config.DatabaseConfig{}, fmt.Errorf("the maintenance database %q has been migrated (goose_db_version "+
			"exists), so this looks like an application database such as the local Compose stack's; refusing "+
			"to create roles on this server", server.Name)
	}
	var superuser bool
	if err := db.Pool().QueryRow(ctx,
		`SELECT rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&superuser); err != nil {
		return config.DatabaseConfig{}, fmt.Errorf("inspect connecting role: %w", err)
	}
	if !superuser {
		return config.DatabaseConfig{}, fmt.Errorf("DB_USER must be the disposable server's superuser: the tests " +
			"create NOLOGIN roles (including one with BYPASSRLS) and switch to them with SET ROLE")
	}
	return server, nil
}

func leftoverRoles(server config.DatabaseConfig) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.MaxConns, server.MinConns = 1, 0
	db, err := database.Open(ctx, server, "sms-rlseval")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Pool().Query(ctx,
		`SELECT rolname FROM pg_roles WHERE starts_with(rolname, $1) ORDER BY rolname`, rolePrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
