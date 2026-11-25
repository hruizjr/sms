//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
)

// databaseExists reports whether a database named name exists, using the maintenance connection.
func databaseExists(t *testing.T, server config.DatabaseConfig, name string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := database.Open(ctx, server, "sms-integration-test")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()
	var exists bool
	if err := db.Pool().QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		t.Fatalf("query pg_database: %v", err)
	}
	return exists
}

// TestIntegrationNewDatabaseLifecycle runs with APP_ENV=test and a loopback DB_HOST (as in CI and
// the documented local setup): the throwaway database is created, used, and dropped, while the
// configured maintenance database is left untouched.
func TestIntegrationNewDatabaseLifecycle(t *testing.T) {
	server := ServerConfig(t)
	if !config.IsLoopbackIP(server.Host) {
		t.Fatalf("ServerConfig() returned non-loopback host %q", server.Host)
	}

	var created string
	t.Run("create", func(t *testing.T) {
		cfg := NewDatabase(t)
		created = cfg.Name
		if !strings.HasPrefix(created, NamePrefix) || created == server.Name {
			t.Fatalf("NewDatabase() name = %q, want a generated %s name distinct from %q", created, NamePrefix, server.Name)
		}
		if !databaseExists(t, server, created) {
			t.Fatalf("database %q was not created", created)
		}
	})

	// The subtest's cleanup has run: the throwaway database is gone, the maintenance database is not.
	if databaseExists(t, server, created) {
		t.Errorf("database %q still exists after cleanup", created)
	}
	if !databaseExists(t, server, server.Name) {
		t.Errorf("the maintenance database %q no longer exists", server.Name)
	}
}
