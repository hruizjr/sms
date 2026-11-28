//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database/dbroles"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
)

// lookupFor returns the process environment with DB_NAME pointing at the throwaway database, the
// given role and password, and APP_ENV=test.
func lookupFor(cfg config.DatabaseConfig) config.LookupFunc {
	return func(key string) (string, bool) {
		switch key {
		case config.EnvDBName:
			return cfg.Name, true
		case config.EnvDBUser:
			return cfg.User, true
		case config.EnvDBPassword:
			return cfg.Password.Reveal(), true
		case config.EnvAppEnv:
			return "test", true
		}
		return os.LookupEnv(key)
	}
}

// TestIntegrationMigrateCommands drives the migrate command end to end against an isolated database,
// connected as the migration role.
func TestIntegrationMigrateCommands(t *testing.T) {
	t.Parallel()
	cfg := dbtest.RoleConfig(t, dbtest.NewDatabase(t), dbroles.Migrator)
	secret := cfg.Password.Reveal()
	lookup := lookupFor(cfg)
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
		{"status", exitOK, []string{"VERSION", "20261125041208", "pending", "20261125041208_baseline.sql",
			"20261126175603_create_tenants.sql", "20261127150000_create_messaging.sql", "20261127200000_create_portal_accounts.sql",
			"20261128120000_create_security_audit_and_sign_in_limits.sql"}},
		{"up", exitOK, []string{"20261125041208_baseline.sql", "20261126175603_create_tenants.sql",
			"20261127150000_create_messaging.sql", "20261127200000_create_portal_accounts.sql",
			"20261128120000_create_security_audit_and_sign_in_limits.sql", "database version: 20261128120000"}},
		{"up", exitOK, []string{"no pending migrations", "database version: 20261128120000"}},
		{"status", exitOK, []string{"applied"}},
		{"version", exitOK, []string{"20261128120000"}},
		{"down", exitOK, []string{"20261128120000_create_security_audit_and_sign_in_limits.sql", "database version: 20261127200000"}},
		{"down", exitOK, []string{"20261127200000_create_portal_accounts.sql", "database version: 20261127150000"}},
		{"down", exitOK, []string{"20261127150000_create_messaging.sql", "database version: 20261126175603"}},
		{"down", exitOK, []string{"20261126175603_create_tenants.sql", "database version: 20261125041208"}},
		{"down", exitOK, []string{"20261125041208_baseline.sql", "database version: 0"}},
		{"down", exitError, nil},
		{"up", exitOK, []string{"database version: 20261128120000"}},
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

// TestIntegrationMigrateRefusesPrivilegedRoles: every command is refused, before any migration
// runs, when the connected role is the bootstrap superuser, and the message says why.
func TestIntegrationMigrateRefusesPrivilegedRoles(t *testing.T) {
	t.Parallel()
	admin := dbtest.NewDatabase(t)
	for _, command := range []string{"up", "status", "version", "down"} {
		var stdout, stderr bytes.Buffer
		code := run(context.Background(), []string{command}, lookupFor(admin), &stdout, &stderr)
		if code != exitError {
			t.Errorf("migrate %s as the superuser: exit code %d, want %d", command, code, exitError)
		}
		if !strings.Contains(stderr.String(), "as the migration role: is a superuser") {
			t.Errorf("migrate %s as the superuser: stderr does not explain the refusal:\n%s", command, stderr.String())
		}
		if strings.Contains(stdout.String()+stderr.String(), admin.Password.Reveal()) {
			t.Fatalf("migrate %s printed the database password", command)
		}
	}
	// Nothing was applied: goose's version table was never even created.
	app := dbtest.RoleConfig(t, admin, dbroles.Migrator)
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"version"}, lookupFor(app), &stdout, &stderr); code != exitOK ||
		strings.TrimSpace(stdout.String()) != "0" {
		t.Errorf("version as the migration role afterwards = %d %q, want 0 \"0\"", code, stdout.String())
	}
}
