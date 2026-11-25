package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// env returns a LookupFunc backed by a map, so tests never read the real process environment.
func env(vars map[string]string) func(string) (string, bool) {
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

func TestRunRefusesDownInProduction(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	// No database is reachable at this address; the refusal must happen before connecting.
	code := run(context.Background(), []string{"down"}, env(map[string]string{
		"APP_ENV":     "production",
		"DB_PASSWORD": "synthetic",
		"DB_SSLMODE":  "require",
		"DB_PORT":     "1",
	}), &stdout, &stderr)
	if code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), "forward-only") {
		t.Errorf("stderr should explain the refusal: %q", stderr.String())
	}
}
