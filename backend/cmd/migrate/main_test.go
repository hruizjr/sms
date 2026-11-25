package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hruizjr/sms/backend/internal/config"
)

// env returns a LookupFunc backed by a map, so tests never read the real process environment.
func env(vars map[string]string) config.LookupFunc {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func TestRunUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{"no command", nil, exitUsage, "", "Usage: migrate"},
		{"two commands", []string{"up", "down"}, exitUsage, "", "Usage: migrate"},
		{"unknown command", []string{"reset"}, exitUsage, "", `unknown command "reset"`},
		{"help", []string{"help"}, exitOK, "Usage: migrate", ""},
		{"--help", []string{"--help"}, exitOK, "Usage: migrate", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), tt.args, env(nil), &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) || !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunInvalidConfigurationNeverPrintsPassword(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-migrate-secret"
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"status"}, env(map[string]string{
		"DB_PASSWORD": secret,
		"DB_PORT":     "not-a-port",
	}), &stdout, &stderr)
	if code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), "DB_PORT") {
		t.Errorf("stderr should name the invalid variable: %q", stderr.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), secret) {
		t.Error("output reveals the password")
	}
}

func TestCheckRollbackAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appEnv  string // "<unset>" leaves APP_ENV unset
		host    string
		allowed bool
		wantMsg string
	}{
		{"development, loopback IPv4", "development", "127.0.0.1", true, ""},
		{"test, loopback IPv4", "test", "127.0.0.1", true, ""},
		{"test, loopback 127/8", "test", "127.0.0.2", true, ""},
		{"development, loopback IPv6", "development", "::1", true, ""},
		{"APP_ENV case and space folded", " Development ", "127.0.0.1", true, ""},

		{"APP_ENV missing", "<unset>", "127.0.0.1", false, "APP_ENV is not set"},
		{"APP_ENV blank", "  ", "127.0.0.1", false, "APP_ENV is not set"},
		{"production, loopback", "production", "127.0.0.1", false, "forward-only"},
		{"production, upper case", "PRODUCTION", "127.0.0.1", false, "forward-only"},
		{"production, remote", "production", "10.0.0.5", false, "forward-only"},
		{"unknown environment", "staging", "127.0.0.1", false, `APP_ENV="staging"`},
		{"development, localhost", "development", "localhost", false, `DB_HOST="localhost"`},
		{"test, remote hostname", "test", "db.example.com", false, `DB_HOST="db.example.com"`},
		{"development, private IPv4", "development", "10.0.0.5", false, `DB_HOST="10.0.0.5"`},
		{"test, public IPv4", "test", "192.0.2.10", false, `DB_HOST="192.0.2.10"`},
		{"test, wildcard IPv4", "test", "0.0.0.0", false, `DB_HOST="0.0.0.0"`},
		{"development, non-loopback IPv6", "development", "2001:db8::1", false, `DB_HOST="2001:db8::1"`},
		{"test, IPv4-mapped non-loopback", "test", "::ffff:10.0.0.5", false, `DB_HOST="::ffff:10.0.0.5"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{}
			if tt.appEnv != "<unset>" {
				vars[config.EnvAppEnv] = tt.appEnv
			}
			err := checkRollbackAllowed(env(vars), config.DatabaseConfig{Host: tt.host})
			if tt.allowed {
				if err != nil {
					t.Errorf("checkRollbackAllowed() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, errRollbackRefused) {
				t.Fatalf("checkRollbackAllowed() = %v, want errRollbackRefused", err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error %q should mention %q", err, tt.wantMsg)
			}
		})
	}
}

// runWith runs one command with a password and the given variables. DB_PORT=1 on 127.0.0.1 has
// nothing listening, so a command that passes the rollback gate fails fast at "ping database".
func runWith(t *testing.T, command string, vars map[string]string) (int, string) {
	t.Helper()
	all := map[string]string{"DB_PASSWORD": "synthetic-migrate-password", "DB_PORT": "1", "DB_CONNECT_TIMEOUT": "500ms"}
	for k, v := range vars {
		all[k] = v
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{command}, env(all), &stdout, &stderr)
	out := stdout.String() + stderr.String()
	if strings.Contains(out, "synthetic-migrate-password") {
		t.Errorf("migrate %s printed the password", command)
	}
	return code, out
}

func TestRunDownRefusedBeforeConnecting(t *testing.T) {
	t.Parallel()

	for name, vars := range map[string]map[string]string{
		"APP_ENV missing":         {"DB_HOST": "127.0.0.1"},
		"APP_ENV production":      {"APP_ENV": "production", "DB_HOST": "127.0.0.1", "DB_SSLMODE": "require"},
		"development, localhost":  {"APP_ENV": "development", "DB_HOST": "localhost"},
		"test, remote hostname":   {"APP_ENV": "test", "DB_HOST": "db.example.com"},
		"development, remote IP":  {"APP_ENV": "development", "DB_HOST": "192.0.2.10"},
		"test, non-loopback IPv6": {"APP_ENV": "test", "DB_HOST": "2001:db8::1"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			code, out := runWith(t, "down", vars)
			if code != exitError {
				t.Errorf("exit code = %d, want %d", code, exitError)
			}
			if !strings.Contains(out, "down is refused") {
				t.Errorf("output should explain the refusal: %q", out)
			}
			if strings.Contains(out, "ping database") || strings.Contains(out, "migrate starting") {
				t.Errorf("down must be refused before connecting: %q", out)
			}
		})
	}
}

func TestRunDownAllowedLocallyReachesDatabase(t *testing.T) {
	t.Parallel()

	for name, vars := range map[string]map[string]string{
		"development, 127.0.0.1": {"APP_ENV": "development", "DB_HOST": "127.0.0.1"},
		"test, 127.0.0.1":        {"APP_ENV": "test", "DB_HOST": "127.0.0.1"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			code, out := runWith(t, "down", vars)
			if strings.Contains(out, "down is refused") {
				t.Fatalf("down was refused for an allowed local setup: %q", out)
			}
			// Nothing listens on port 1, so the command gets as far as the database ping and fails there.
			if code != exitError || !strings.Contains(out, "ping database") {
				t.Errorf("exit %d, output %q; want the gate passed and a ping failure", code, out)
			}
		})
	}
}

func TestRunOtherCommandsUnaffectedByRollbackRules(t *testing.T) {
	t.Parallel()

	// up, status, and version keep their previous behaviour: no APP_ENV or host restriction.
	for _, command := range []string{"up", "status", "version"} {
		for name, vars := range map[string]map[string]string{
			"APP_ENV missing":    {"DB_HOST": "127.0.0.1"},
			"APP_ENV production": {"APP_ENV": "production", "DB_HOST": "127.0.0.1", "DB_SSLMODE": "require"},
		} {
			t.Run(command+"/"+name, func(t *testing.T) {
				t.Parallel()
				code, out := runWith(t, command, vars)
				if strings.Contains(out, "refused") && !strings.Contains(out, "connection refused") {
					t.Errorf("%s must not be subject to the rollback rules: %q", command, out)
				}
				if code != exitError || !strings.Contains(out, "ping database") {
					t.Errorf("exit %d, output %q; want a database ping attempt", code, out)
				}
			})
		}
	}
}
