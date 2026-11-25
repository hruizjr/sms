package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// env returns a LookupFunc backed by a map, so tests never read the real process environment.
func env(vars map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() with no variables returned error: %v", err)
	}

	want := Config{
		Env: EnvironmentDevelopment,
		HTTP: HTTPConfig{
			Addr:              DefaultHTTPAddr,
			ReadTimeout:       DefaultHTTPReadTimeout,
			ReadHeaderTimeout: DefaultHTTPReadHeaderTimeout,
			WriteTimeout:      DefaultHTTPWriteTimeout,
			IdleTimeout:       DefaultHTTPIdleTimeout,
		},
		Metrics:         MetricsConfig{Addr: DefaultMetricsAddr},
		Log:             LogConfig{Format: LogFormatText, Level: slog.LevelInfo},
		ShutdownTimeout: DefaultShutdownTimeout,
	}
	if cfg != want {
		t.Errorf("Load() defaults:\n got %+v\nwant %+v", cfg, want)
	}
}

func TestLoadBlankValuesUseDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{EnvHTTPAddr: "   ", EnvLogLevel: ""}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.HTTP.Addr != DefaultHTTPAddr || cfg.Log.Level != slog.LevelInfo {
		t.Errorf("blank values should fall back to defaults, got addr=%q level=%v", cfg.HTTP.Addr, cfg.Log.Level)
	}
}

func TestLoadValidCustomValues(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{
		EnvAppEnv:                "Production",
		EnvHTTPAddr:              ":8081",
		EnvHTTPReadTimeout:       "20s",
		EnvHTTPReadHeaderTimeout: "2s",
		EnvHTTPWriteTimeout:      "30s",
		EnvHTTPIdleTimeout:       "2m",
		EnvShutdownTimeout:       "45s",
		EnvMetricsAddr:           "[::1]:9191",
		EnvLogFormat:             "TEXT",
		EnvLogLevel:              "debug",
	}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	want := Config{
		Env: EnvironmentProduction,
		HTTP: HTTPConfig{
			Addr:              ":8081",
			ReadTimeout:       20 * time.Second,
			ReadHeaderTimeout: 2 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       2 * time.Minute,
		},
		Metrics:         MetricsConfig{Addr: "[::1]:9191"},
		Log:             LogConfig{Format: LogFormatText, Level: slog.LevelDebug},
		ShutdownTimeout: 45 * time.Second,
	}
	if cfg != want {
		t.Errorf("Load() custom values:\n got %+v\nwant %+v", cfg, want)
	}
}

func TestLoadDefaultLogFormatDependsOnEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		appEnv string
		want   LogFormat
	}{
		{"development", LogFormatText},
		{"test", LogFormatJSON},
		{"production", LogFormatJSON},
	}
	for _, tt := range tests {
		t.Run(tt.appEnv, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(env(map[string]string{EnvAppEnv: tt.appEnv}))
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}
			if cfg.Log.Format != tt.want {
				t.Errorf("default log format for %s = %q, want %q", tt.appEnv, cfg.Log.Format, tt.want)
			}
		})
	}
}

func TestLoadMetricsAddrAcceptsLoopbackIPs(t *testing.T) {
	t.Parallel()

	for _, addr := range []string{
		"127.0.0.1:9090",          // IPv4 loopback (the default)
		"127.0.0.2:9090",          // anywhere in 127.0.0.0/8 is loopback
		"[::1]:9090",              // IPv6 loopback
		"[::ffff:127.0.0.1]:9090", // IPv4-mapped IPv6 form of 127.0.0.1
	} {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(env(map[string]string{EnvMetricsAddr: addr}))
			if err != nil {
				t.Fatalf("Load() with %s=%q returned error: %v", EnvMetricsAddr, addr, err)
			}
			if cfg.Metrics.Addr != addr {
				t.Errorf("Metrics.Addr = %q, want %q", cfg.Metrics.Addr, addr)
			}
		})
	}
}

func TestLoadMetricsAddrRejectsNonLoopback(t *testing.T) {
	t.Parallel()

	for _, addr := range []string{
		":9090",                  // empty host: all interfaces
		"0.0.0.0:9090",           // IPv4 wildcard
		"[::]:9090",              // IPv6 wildcard
		"192.0.2.10:9090",        // non-loopback IPv4 (TEST-NET-1, RFC 5737)
		"10.0.0.5:9090",          // private IPv4
		"[2001:db8::1]:9090",     // non-loopback IPv6 (documentation prefix, RFC 3849)
		"[fe80::1]:9090",         // IPv6 link-local
		"[::ffff:10.0.0.5]:9090", // IPv4-mapped non-loopback
		"localhost:9090",         // host names are not resolved during validation, so not accepted
	} {
		t.Run(addr, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(env(map[string]string{EnvMetricsAddr: addr}))
			if err == nil {
				t.Fatalf("Load() with %s=%q = %+v, want error", EnvMetricsAddr, addr, cfg)
			}
			if !strings.Contains(err.Error(), EnvMetricsAddr) || !strings.Contains(err.Error(), "loopback") {
				t.Errorf("error %q should name %s and require a loopback address", err, EnvMetricsAddr)
			}
		})
	}
}

func TestLoadIgnoresRemovedMetricsOverride(t *testing.T) {
	t.Parallel()

	// METRICS_ALLOW_NON_LOOPBACK was removed; setting it must not re-enable a public listener.
	_, err := Load(env(map[string]string{
		EnvMetricsAddr:               "0.0.0.0:9090",
		"METRICS_ALLOW_NON_LOOPBACK": "true",
	}))
	if err == nil || !strings.Contains(err.Error(), EnvMetricsAddr) {
		t.Fatalf("0.0.0.0 with the removed override set: want %s error, got %v", EnvMetricsAddr, err)
	}
}

func TestLoadInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		vars    map[string]string
		wantKey string
	}{
		{"unknown environment", map[string]string{EnvAppEnv: "staging"}, EnvAppEnv},
		{"address without port", map[string]string{EnvHTTPAddr: "8080"}, EnvHTTPAddr},
		{"address port out of range", map[string]string{EnvHTTPAddr: "127.0.0.1:70000"}, EnvHTTPAddr},
		{"address port zero", map[string]string{EnvHTTPAddr: "127.0.0.1:0"}, EnvHTTPAddr},
		{"address non-numeric port", map[string]string{EnvHTTPAddr: "127.0.0.1:http"}, EnvHTTPAddr},
		{"address hostname", map[string]string{EnvHTTPAddr: "api.internal:8080"}, EnvHTTPAddr},
		{"metrics address invalid", map[string]string{EnvMetricsAddr: "nonsense"}, EnvMetricsAddr},
		{"metrics on all interfaces", map[string]string{EnvMetricsAddr: ":9090"}, EnvMetricsAddr},
		{"metrics same listener as http", map[string]string{EnvHTTPAddr: "127.0.0.1:8080", EnvMetricsAddr: "127.0.0.1:8080"}, EnvMetricsAddr},
		{"metrics port overlaps all-interfaces http", map[string]string{EnvHTTPAddr: ":9090", EnvMetricsAddr: "127.0.0.1:9090"}, EnvMetricsAddr},
		{"timeout not a duration", map[string]string{EnvHTTPReadTimeout: "fifteen"}, EnvHTTPReadTimeout},
		{"timeout without unit", map[string]string{EnvHTTPWriteTimeout: "15"}, EnvHTTPWriteTimeout},
		{"timeout zero", map[string]string{EnvHTTPIdleTimeout: "0s"}, EnvHTTPIdleTimeout},
		{"timeout negative", map[string]string{EnvShutdownTimeout: "-5s"}, EnvShutdownTimeout},
		{"header timeout above read timeout", map[string]string{EnvHTTPReadTimeout: "5s", EnvHTTPReadHeaderTimeout: "10s"}, EnvHTTPReadHeaderTimeout},
		{"log level unknown", map[string]string{EnvLogLevel: "verbose"}, EnvLogLevel},
		{"log format unknown", map[string]string{EnvLogFormat: "xml"}, EnvLogFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(env(tt.vars))
			if err == nil {
				t.Fatalf("Load() = %+v, want error", cfg)
			}
			if !strings.Contains(err.Error(), tt.wantKey) {
				t.Errorf("error %q does not name the offending variable %s", err, tt.wantKey)
			}
			if cfg != (Config{}) {
				t.Errorf("Load() returned a partially filled Config on error: %+v", cfg)
			}
		})
	}
}

func TestLoadReportsAllErrors(t *testing.T) {
	t.Parallel()

	_, err := Load(env(map[string]string{
		EnvHTTPAddr:  "bad",
		EnvLogLevel:  "loud",
		EnvLogFormat: "yaml",
	}))
	if err == nil {
		t.Fatal("Load() returned nil error")
	}
	for _, key := range []string{EnvHTTPAddr, EnvLogLevel, EnvLogFormat} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}
