//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hruizjr/sms/backend/internal/database/dbtest"
)

// TestIntegrationMigrateCommands drives the migrate command end to end against an isolated database.
func TestIntegrationMigrateCommands(t *testing.T) {
	t.Parallel()
	cfg := dbtest.NewDatabase(t)
	secret := cfg.Password.Reveal()

	// The process environment, with DB_NAME pointing at the throwaway database.
	lookup := func(key string) (string, bool) {
		if key == "DB_NAME" {
			return cfg.Name, true
		}
		if key == "APP_ENV" {
			return "test", true
		}
		return os.LookupEnv(key)
	}
	runCmd := func(command string) (int, string) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		code := run(context.Background(), []string{command}, lookup, &stdout, &stderr)
		if strings.Contains(stdout.String()+stderr.String(), secret) {
			t.Fatalf("migrate %s printed the database password", command)
		}
		if code != exitOK {
			t.Logf("migrate %s stderr: %s", command, stderr.String())
		}
		return code, stdout.String()
	}

	steps := []struct {
		command  string
		wantCode int
		wantOut  []string
	}{
		{"version", exitOK, []string{"0"}},
		{"status", exitOK, []string{"VERSION", "20261125041208", "pending", "20261125041208_baseline.sql"}},
		{"up", exitOK, []string{"20261125041208_baseline.sql", "database version: 20261125041208"}},
		{"up", exitOK, []string{"no pending migrations", "database version: 20261125041208"}},
		{"status", exitOK, []string{"applied"}},
		{"version", exitOK, []string{"20261125041208"}},
		{"down", exitOK, []string{"20261125041208_baseline.sql", "database version: 0"}},
		{"down", exitError, nil},
		{"up", exitOK, []string{"database version: 20261125041208"}},
	}
	for i, s := range steps {
		code, out := runCmd(s.command)
		if code != s.wantCode {
			t.Fatalf("step %d: migrate %s exit code = %d, want %d; output: %s", i+1, s.command, code, s.wantCode, out)
		}
		for _, want := range s.wantOut {
			if !strings.Contains(out, want) {
				t.Errorf("step %d: migrate %s output missing %q:\n%s", i+1, s.command, want, out)
			}
		}
	}
}
