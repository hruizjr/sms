package dbtest

import (
	"errors"
	"strings"
	"testing"

	"github.com/hruizjr/sms/backend/internal/config"
)

// syntheticPassword is not a real credential; tests assert it never appears in errors.
const syntheticPassword = "synthetic-dbtest-password"

// env returns a LookupFunc for a local test setup (APP_ENV=test, DB_HOST=127.0.0.1, a password),
// with vars applied on top. A key mapped to "<unset>" is removed.
func env(vars map[string]string) config.LookupFunc {
	base := map[string]string{
		config.EnvAppEnv:     "test",
		config.EnvDBHost:     "127.0.0.1",
		config.EnvDBPassword: syntheticPassword,
	}
	for k, v := range vars {
		if v == "<unset>" {
			delete(base, k)
			continue
		}
		base[k] = v
	}
	return func(key string) (string, bool) {
		v, ok := base[key]
		return v, ok
	}
}

func TestTargetConfigAcceptsLocalTestSetup(t *testing.T) {
	t.Parallel()

	for name, vars := range map[string]map[string]string{
		"loopback IPv4":                 nil,
		"loopback IPv4 127/8":           {config.EnvDBHost: "127.0.0.2"},
		"loopback IPv6":                 {config.EnvDBHost: "::1"},
		"DB_HOST unset uses 127.0.0.1":  {config.EnvDBHost: "<unset>"},
		"APP_ENV case and space folded": {config.EnvAppEnv: " Test "},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg, err := TargetConfig(env(vars))
			if err != nil {
				t.Fatalf("TargetConfig() error: %v", err)
			}
			if !config.IsLoopbackIP(cfg.Host) {
				t.Errorf("Host = %q, want a loopback IP", cfg.Host)
			}
		})
	}
}

func TestTargetConfigRefusesUnsafeSetups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		vars    map[string]string
		wantMsg string
	}{
		{"APP_ENV missing", map[string]string{config.EnvAppEnv: "<unset>"}, "APP_ENV is not set"},
		{"APP_ENV blank", map[string]string{config.EnvAppEnv: "  "}, "APP_ENV is not set"},
		{"APP_ENV development", map[string]string{config.EnvAppEnv: "development"}, `APP_ENV="development"`},
		{"APP_ENV production", map[string]string{config.EnvAppEnv: "production"}, `APP_ENV="production"`},
		{"APP_ENV unknown", map[string]string{config.EnvAppEnv: "staging"}, `APP_ENV="staging"`},
		{"non-loopback IPv4", map[string]string{config.EnvDBHost: "10.0.0.5"}, `DB_HOST="10.0.0.5"`},
		{"public IPv4", map[string]string{config.EnvDBHost: "192.0.2.10"}, `DB_HOST="192.0.2.10"`},
		{"wildcard IPv4", map[string]string{config.EnvDBHost: "0.0.0.0"}, `DB_HOST="0.0.0.0"`},
		{"non-loopback IPv6", map[string]string{config.EnvDBHost: "2001:db8::1"}, `DB_HOST="2001:db8::1"`},
		{"IPv4-mapped non-loopback", map[string]string{config.EnvDBHost: "::ffff:10.0.0.5"}, `DB_HOST="::ffff:10.0.0.5"`},
		{"hostname localhost", map[string]string{config.EnvDBHost: "localhost"}, `DB_HOST="localhost"`},
		{"remote hostname", map[string]string{config.EnvDBHost: "db.example.com"}, `DB_HOST="db.example.com"`},
		{"missing password", map[string]string{config.EnvDBPassword: "<unset>"}, "DB_PASSWORD"},
		{"invalid port", map[string]string{config.EnvDBPort: "not-a-port"}, "DB_PORT"},
		{"invalid sslmode", map[string]string{config.EnvDBSSLMode: "prefer"}, "DB_SSLMODE"},
		{"invalid pool size", map[string]string{config.EnvDBMaxConns: "0"}, "DB_MAX_CONNS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := TargetConfig(env(tt.vars))
			if err == nil {
				t.Fatalf("TargetConfig() = %+v, want an error", cfg)
			}
			if !errors.Is(err, ErrUnsafeTarget) {
				t.Errorf("error %v does not wrap ErrUnsafeTarget", err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error %q should mention %q", err, tt.wantMsg)
			}
			if strings.Contains(err.Error(), syntheticPassword) {
				t.Errorf("error reveals the password: %v", err)
			}
			if cfg != (config.DatabaseConfig{}) {
				t.Errorf("TargetConfig() returned settings with an error: %+v", cfg)
			}
		})
	}
}

func TestCheckDisposableName(t *testing.T) {
	t.Parallel()

	accepted := []string{"sms_test_0123456789abcdef", "sms_test_missing_ab12"}
	for _, name := range accepted {
		if err := checkDisposableName(name, "sms"); err != nil {
			t.Errorf("checkDisposableName(%q) = %v, want nil", name, err)
		}
	}

	refused := []struct{ name, maintenance string }{
		{"sms", "sms"},                                 // the maintenance database itself
		{"sms_test_abc", "sms_test_abc"},               // a maintenance database that happens to have the prefix
		{"postgres", "sms"},                            // no prefix
		{"sms_test_", "sms"},                           // prefix only
		{"sms_test_ABC", "sms"},                        // outside [a-z0-9_]
		{`sms_test_a"; DROP x; --`, "sms"},             // injection-shaped
		{"sms_test_" + strings.Repeat("a", 60), "sms"}, // longer than 63 bytes
	}
	for _, tt := range refused {
		if err := checkDisposableName(tt.name, tt.maintenance); err == nil {
			t.Errorf("checkDisposableName(%q, %q) = nil, want an error", tt.name, tt.maintenance)
		}
	}
}
