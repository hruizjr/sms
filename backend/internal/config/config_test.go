package config

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// testPassword is a synthetic database password used only by these tests.
const testPassword = "synthetic-test-password"

// env returns a LookupFunc backed by a map, so tests never read the real process environment.
// DB_PASSWORD (required, no default) is set to testPassword unless vars sets it explicitly.
func env(vars map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		if v, ok := vars[key]; ok {
			return v, ok
		}
		if key == EnvDBPassword {
			return testPassword, true
		}
		return "", false
	}
}

// defaultDatabase is the database configuration produced by the defaults for env.
func defaultDatabase(sslMode SSLMode) DatabaseConfig {
	return DatabaseConfig{
		Host:            DefaultDBHost,
		Port:            DefaultDBPort,
		Name:            DefaultDBName,
		User:            DefaultDBUser,
		Password:        Secret(testPassword),
		SSLMode:         sslMode,
		MaxConns:        DefaultDBMaxConns,
		MinConns:        DefaultDBMinConns,
		ConnectTimeout:  DefaultDBConnectTimeout,
		MaxConnLifetime: DefaultDBMaxConnLifetime,
		MaxConnIdleTime: DefaultDBMaxConnIdleTime,
	}
}

func defaultProvider(name string) ProviderConfig {
	return ProviderConfig{Name: name, Simulator: SimulatorConfig{ReceiptDelay: DefaultSimulatorReceiptDelay}}
}

func defaultDispatch() DispatchConfig {
	return DispatchConfig{
		PollInterval:   DefaultDispatchPollInterval,
		BatchSize:      DefaultDispatchBatchSize,
		Concurrency:    DefaultDispatchConcurrency,
		SubmitTimeout:  DefaultDispatchSubmitTimeout,
		Lease:          DefaultDispatchLease,
		MaxAttempts:    DefaultDispatchMaxAttempts,
		RetryBaseDelay: DefaultDispatchRetryBaseDelay,
		RetryMaxDelay:  DefaultDispatchRetryMaxDelay,
		MaxMessageAge:  DefaultDispatchMaxMessageAge,
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() with only DB_PASSWORD set returned error: %v", err)
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
		Database:        defaultDatabase(SSLModeDisable),
		ShutdownTimeout: DefaultShutdownTimeout,
		Provider:        defaultProvider(ProviderSimulator),
		Dispatch:        defaultDispatch(),
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
		Database:        defaultDatabase(SSLModeVerifyFull),
		ShutdownTimeout: 45 * time.Second,
		// Production has no default provider (no production provider exists yet).
		Provider: defaultProvider(""),
		Dispatch: defaultDispatch(),
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
		{"db password missing", map[string]string{EnvDBPassword: ""}, EnvDBPassword},
		{"db host with space", map[string]string{EnvDBHost: "db host"}, EnvDBHost},
		{"db host list", map[string]string{EnvDBHost: "db1,db2"}, EnvDBHost},
		{"db port not a number", map[string]string{EnvDBPort: "five"}, EnvDBPort},
		{"db port out of range", map[string]string{EnvDBPort: "70000"}, EnvDBPort},
		{"db name too long", map[string]string{EnvDBName: strings.Repeat("n", 64)}, EnvDBName},
		{"db user control character", map[string]string{EnvDBUser: "sms\x00"}, EnvDBUser},
		{"db sslmode prefer", map[string]string{EnvDBSSLMode: "prefer"}, EnvDBSSLMode},
		{"db sslmode unknown", map[string]string{EnvDBSSLMode: "on"}, EnvDBSSLMode},
		{"db sslmode disable in production", map[string]string{EnvAppEnv: "production", EnvDBSSLMode: "disable"}, EnvDBSSLMode},
		{"db max conns zero", map[string]string{EnvDBMaxConns: "0"}, EnvDBMaxConns},
		{"db max conns above limit", map[string]string{EnvDBMaxConns: "101"}, EnvDBMaxConns},
		{"db min conns above max", map[string]string{EnvDBMaxConns: "4", EnvDBMinConns: "5"}, EnvDBMinConns},
		{"db connect timeout zero", map[string]string{EnvDBConnectTimeout: "0s"}, EnvDBConnectTimeout},
		{"db max conn lifetime not a duration", map[string]string{EnvDBMaxConnLifetime: "1 hour"}, EnvDBMaxConnLifetime},
		{"db max conn idle time negative", map[string]string{EnvDBMaxConnIdleTime: "-1m"}, EnvDBMaxConnIdleTime},
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

func TestLoadDatabaseCustomValues(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{
		EnvDBHost:            "db.internal.example",
		EnvDBPort:            "6543",
		EnvDBName:            "sms_dev",
		EnvDBUser:            "sms_app",
		EnvDBPassword:        " pass with spaces 'and' \\ backslash ",
		EnvDBSSLMode:         "VERIFY-CA",
		EnvDBMaxConns:        "25",
		EnvDBMinConns:        "2",
		EnvDBConnectTimeout:  "3s",
		EnvDBMaxConnLifetime: "45m",
		EnvDBMaxConnIdleTime: "5m",
	}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	want := DatabaseConfig{
		Host:            "db.internal.example",
		Port:            6543,
		Name:            "sms_dev",
		User:            "sms_app",
		Password:        Secret(" pass with spaces 'and' \\ backslash "),
		SSLMode:         SSLModeVerifyCA,
		MaxConns:        25,
		MinConns:        2,
		ConnectTimeout:  3 * time.Second,
		MaxConnLifetime: 45 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
	}
	if cfg.Database != want {
		t.Errorf("Database config:\n got %#v\nwant %#v", cfg.Database, want)
	}
	if cfg.Database.Password.Reveal() != want.Password.Reveal() {
		t.Error("password must be read verbatim, including leading and trailing spaces")
	}
}

func TestLoadDatabaseSSLModeDefaultsByEnvironment(t *testing.T) {
	t.Parallel()

	for appEnv, want := range map[string]SSLMode{
		"development": SSLModeDisable,
		"test":        SSLModeDisable,
		"production":  SSLModeVerifyFull,
	} {
		t.Run(appEnv, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(env(map[string]string{EnvAppEnv: appEnv}))
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}
			if cfg.Database.SSLMode != want {
				t.Errorf("default DB_SSLMODE for %s = %q, want %q", appEnv, cfg.Database.SSLMode, want)
			}
		})
	}
}

func TestLoadDatabasePasswordNeverInErrors(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-secret-in-error-test"
	_, err := Load(env(map[string]string{
		EnvDBPassword: secret,
		EnvDBPort:     "not-a-port",
		EnvDBSSLMode:  "bogus",
		EnvDBHost:     "bad host",
	}))
	if err == nil {
		t.Fatal("Load() returned nil error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("configuration error reveals the password: %v", err)
	}
}

func TestSecretIsRedactedWhenPrinted(t *testing.T) {
	t.Parallel()

	const secret = "synthetic-secret-print-test"
	cfg, err := Load(env(map[string]string{EnvDBPassword: secret}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	var jsonLogs, textLogs strings.Builder
	slog.New(slog.NewJSONHandler(&jsonLogs, nil)).Info("config", "database", cfg.Database, "password", cfg.Database.Password)
	slog.New(slog.NewTextHandler(&textLogs, nil)).Info("config", "database", cfg.Database, "password", cfg.Database.Password)

	for name, out := range map[string]string{
		"%v":        fmt.Sprintf("%v", cfg),
		"%+v":       fmt.Sprintf("%+v", cfg),
		"%#v":       fmt.Sprintf("%#v", cfg),
		"String()":  cfg.Database.Password.String(),
		"slog json": jsonLogs.String(),
		"slog text": textLogs.String(),
	} {
		if strings.Contains(out, secret) {
			t.Errorf("%s output reveals the password: %s", name, out)
		}
		if !strings.Contains(out, redacted) {
			t.Errorf("%s output should show %s: %s", name, redacted, out)
		}
	}
	if cfg.Database.Password.Reveal() != secret {
		t.Error("Reveal() must return the secret value")
	}
}

func TestIsLoopbackIP(t *testing.T) {
	t.Parallel()

	for host, want := range map[string]bool{
		"127.0.0.1":        true,
		"127.0.0.2":        true,
		"::1":              true,
		"::ffff:127.0.0.1": true,
		"":                 false,
		"0.0.0.0":          false,
		"::":               false,
		"10.0.0.5":         false,
		"192.0.2.10":       false,
		"2001:db8::1":      false,
		"::ffff:10.0.0.5":  false,
		"localhost":        false,
		"db.example.com":   false,
	} {
		if got := IsLoopbackIP(host); got != want {
			t.Errorf("IsLoopbackIP(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestExplicitEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vars   map[string]string
		want   Environment
		wantOK bool
	}{
		{"missing", map[string]string{}, "", false},
		{"blank", map[string]string{EnvAppEnv: "  "}, "", false},
		{"test", map[string]string{EnvAppEnv: "test"}, EnvironmentTest, true},
		{"normalized", map[string]string{EnvAppEnv: " Development "}, EnvironmentDevelopment, true},
		{"unknown value returned as is", map[string]string{EnvAppEnv: "staging"}, "staging", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ExplicitEnvironment(env(tt.vars))
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ExplicitEnvironment() = %q, %v; want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestLoadMessagingSettings(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{
		EnvSimulatorReceiptDelay:  "250ms",
		EnvSimulatorMaxSegments:   "6",
		EnvSimulatorReportWindow:  "2m",
		EnvDispatchPollInterval:   "1s",
		EnvDispatchBatchSize:      "50",
		EnvDispatchConcurrency:    "8",
		EnvDispatchSubmitTimeout:  "5s",
		EnvDispatchLease:          "30s",
		EnvDispatchMaxAttempts:    "3",
		EnvDispatchRetryBaseDelay: "1s",
		EnvDispatchRetryMaxDelay:  "10s",
		EnvDispatchMaxMessageAge:  "1h",
	}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	wantProvider := ProviderConfig{Name: ProviderSimulator, Simulator: SimulatorConfig{
		ReceiptDelay: 250 * time.Millisecond, MaxSegments: 6, ReportWindow: 2 * time.Minute,
	}}
	wantDispatch := DispatchConfig{
		PollInterval: time.Second, BatchSize: 50, Concurrency: 8, SubmitTimeout: 5 * time.Second,
		Lease: 30 * time.Second, MaxAttempts: 3, RetryBaseDelay: time.Second, RetryMaxDelay: 10 * time.Second,
		MaxMessageAge: time.Hour,
	}
	if cfg.Provider != wantProvider || cfg.Dispatch != wantDispatch {
		t.Errorf("got provider %+v dispatch %+v\nwant provider %+v dispatch %+v", cfg.Provider, cfg.Dispatch, wantProvider, wantDispatch)
	}
	if cfg.Provider.ReportWindow() != 2*time.Minute {
		t.Errorf("ReportWindow() = %s", cfg.Provider.ReportWindow())
	}
}

func TestLoadRejectsInvalidMessagingSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vars map[string]string
		key  string
	}{
		{"simulator in production", map[string]string{EnvAppEnv: "production", EnvSMSProvider: "simulator"}, EnvSMSProvider},
		{"unknown provider", map[string]string{EnvSMSProvider: "acme"}, EnvSMSProvider},
		{"lease not above submit timeout", map[string]string{EnvDispatchLease: "5s", EnvDispatchSubmitTimeout: "5s"}, EnvDispatchLease},
		{"retry base above max", map[string]string{EnvDispatchRetryBaseDelay: "1m", EnvDispatchRetryMaxDelay: "1s"}, EnvDispatchRetryMaxDelay},
		{"negative report window", map[string]string{EnvSimulatorReportWindow: "-1s"}, EnvSimulatorReportWindow},
		{"zero batch", map[string]string{EnvDispatchBatchSize: "0"}, EnvDispatchBatchSize},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(env(tc.vars))
			if err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Errorf("Load() error = %v, want one naming %s", err, tc.key)
			}
		})
	}
}
