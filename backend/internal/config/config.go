// Package config loads and validates the API process configuration from environment variables.
//
// Configuration is read once at startup into a typed Config. Invalid values fail fast with an error
// that names every offending variable. Only the standard library is used (decision log §7).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
)

// Environment variable names. They are documented in backend/README.md and backend/.env.example.
//
//nolint:gosec // G101 false positive: these are environment variable names, not credentials.
const (
	EnvAppEnv                = "APP_ENV"
	EnvHTTPAddr              = "HTTP_ADDR"
	EnvHTTPReadTimeout       = "HTTP_READ_TIMEOUT"
	EnvHTTPReadHeaderTimeout = "HTTP_READ_HEADER_TIMEOUT"
	EnvHTTPWriteTimeout      = "HTTP_WRITE_TIMEOUT"
	EnvHTTPIdleTimeout       = "HTTP_IDLE_TIMEOUT"
	EnvShutdownTimeout       = "SHUTDOWN_TIMEOUT"
	EnvMetricsAddr           = "METRICS_ADDR"
	EnvLogFormat             = "LOG_FORMAT"
	EnvLogLevel              = "LOG_LEVEL"

	EnvDBHost            = "DB_HOST"
	EnvDBPort            = "DB_PORT"
	EnvDBName            = "DB_NAME"
	EnvDBUser            = "DB_USER"
	EnvDBPassword        = "DB_PASSWORD"
	EnvDBSSLMode         = "DB_SSLMODE"
	EnvDBMaxConns        = "DB_MAX_CONNS"
	EnvDBMinConns        = "DB_MIN_CONNS"
	EnvDBConnectTimeout  = "DB_CONNECT_TIMEOUT"
	EnvDBMaxConnLifetime = "DB_MAX_CONN_LIFETIME"
	EnvDBMaxConnIdleTime = "DB_MAX_CONN_IDLE_TIME"
)

// Environment is the deployment environment the process runs in.
type Environment string

// Supported environments.
const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentProduction  Environment = "production"
)

// LogFormat selects the slog handler.
type LogFormat string

// Supported log formats.
const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// Default values used when a variable is unset or empty. They are safe for local development:
// both listeners bind to the loopback interface.
const (
	DefaultHTTPAddr              = "127.0.0.1:8080"
	DefaultHTTPReadTimeout       = 15 * time.Second
	DefaultHTTPReadHeaderTimeout = 5 * time.Second
	DefaultHTTPWriteTimeout      = 15 * time.Second
	DefaultHTTPIdleTimeout       = 60 * time.Second
	DefaultShutdownTimeout       = 15 * time.Second
	DefaultMetricsAddr           = "127.0.0.1:9090"

	DefaultDBHost            = "127.0.0.1"
	DefaultDBPort            = 5432
	DefaultDBName            = "sms"
	DefaultDBUser            = "sms"
	DefaultDBMaxConns        = 10
	DefaultDBMinConns        = 0
	DefaultDBConnectTimeout  = 5 * time.Second
	DefaultDBMaxConnLifetime = time.Hour
	DefaultDBMaxConnIdleTime = 30 * time.Minute

	// MaxDBConns is the upper bound accepted for DB_MAX_CONNS. Every API or migrate process opens
	// its own pool, so the sum across processes must stay below the server's max_connections.
	MaxDBConns = 100
	// maxIdentifierLength is PostgreSQL's limit for database and role names (NAMEDATALEN - 1).
	maxIdentifierLength = 63
)

// SSLMode is the PostgreSQL TLS mode (libpq sslmode). The modes "allow" and "prefer" are not
// accepted because they silently fall back to an unencrypted connection.
type SSLMode string

// Supported TLS modes.
const (
	SSLModeDisable    SSLMode = "disable"
	SSLModeRequire    SSLMode = "require"
	SSLModeVerifyCA   SSLMode = "verify-ca"
	SSLModeVerifyFull SSLMode = "verify-full"
)

// Secret holds a sensitive value. Its String, GoString, LogValue, and MarshalJSON methods return a fixed
// placeholder, so printing or logging a configuration value never reveals the secret. Use
// Reveal only where the value is actually needed (for example, to open a connection).
type Secret string

const redacted = "[REDACTED]"

// Reveal returns the secret value.
func (s Secret) Reveal() string { return string(s) }

// String implements fmt.Stringer without revealing the value.
func (Secret) String() string { return redacted }

// GoString implements fmt.GoStringer without revealing the value.
func (Secret) GoString() string { return redacted }

// LogValue implements slog.LogValuer without revealing the value.
func (Secret) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalJSON implements json.Marshaler without revealing the value. slog's JSON handler encodes
// structs (such as DatabaseConfig) with encoding/json, which does not call LogValue on fields.
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// Config is the validated configuration of the API process.
type Config struct {
	Env             Environment
	HTTP            HTTPConfig
	Metrics         MetricsConfig
	Log             LogConfig
	Database        DatabaseConfig
	ShutdownTimeout time.Duration
}

// HTTPConfig configures the public application HTTP server.
type HTTPConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

// MetricsConfig configures the separate, non-public metrics listener.
//
// Addr is always a loopback IP address (127.0.0.0/8 or ::1). Non-loopback and wildcard
// addresses are rejected: exposing metrics beyond the host is not supported in this phase.
type MetricsConfig struct {
	Addr string
}

// DatabaseConfig configures the PostgreSQL connection pool. It is shared by the api and migrate
// executables.
type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password Secret
	SSLMode  SSLMode
	// MaxConns and MinConns bound the number of pooled connections.
	MaxConns int
	MinConns int
	// ConnectTimeout bounds establishing one connection, including the TLS handshake and
	// authentication.
	ConnectTimeout time.Duration
	// MaxConnLifetime and MaxConnIdleTime recycle connections so that server-side changes (for
	// example a failover or a password rotation) are picked up without a restart.
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// LogConfig configures structured logging.
type LogConfig struct {
	Format LogFormat
	Level  slog.Level
}

// LookupFunc returns the value of an environment variable and whether it is set.
// os.LookupEnv satisfies it; tests pass a map-backed function.
type LookupFunc func(key string) (string, bool)

// Load reads the configuration through lookup, applies defaults, and validates it.
// The returned error lists every invalid variable so that all problems can be fixed at once.
func Load(lookup LookupFunc) (Config, error) {
	r := reader{lookup: lookup}

	cfg := Config{
		Env: Environment(strings.ToLower(r.str(EnvAppEnv, string(EnvironmentDevelopment)))),
		HTTP: HTTPConfig{
			Addr:              r.str(EnvHTTPAddr, DefaultHTTPAddr),
			ReadTimeout:       r.duration(EnvHTTPReadTimeout, DefaultHTTPReadTimeout),
			ReadHeaderTimeout: r.duration(EnvHTTPReadHeaderTimeout, DefaultHTTPReadHeaderTimeout),
			WriteTimeout:      r.duration(EnvHTTPWriteTimeout, DefaultHTTPWriteTimeout),
			IdleTimeout:       r.duration(EnvHTTPIdleTimeout, DefaultHTTPIdleTimeout),
		},
		Metrics: MetricsConfig{
			Addr: r.str(EnvMetricsAddr, DefaultMetricsAddr),
		},
		ShutdownTimeout: r.duration(EnvShutdownTimeout, DefaultShutdownTimeout),
	}

	switch cfg.Env {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentProduction:
	default:
		r.fail(EnvAppEnv, "must be one of development, test, production; got %q", cfg.Env)
	}

	defaultFormat := LogFormatJSON
	if cfg.Env == EnvironmentDevelopment {
		defaultFormat = LogFormatText
	}
	cfg.Log.Format = LogFormat(strings.ToLower(r.str(EnvLogFormat, string(defaultFormat))))
	if cfg.Log.Format != LogFormatText && cfg.Log.Format != LogFormatJSON {
		r.fail(EnvLogFormat, "must be text or json; got %q", cfg.Log.Format)
	}
	cfg.Log.Level = r.logLevel(EnvLogLevel, slog.LevelInfo)
	cfg.Database = r.database(cfg.Env)

	httpHost, httpPort, httpOK := r.address(EnvHTTPAddr, cfg.HTTP.Addr)
	metricsHost, metricsPort, metricsOK := r.address(EnvMetricsAddr, cfg.Metrics.Addr)

	if metricsOK && !isLoopbackIP(metricsHost) {
		r.fail(EnvMetricsAddr, "must be a loopback IP address such as 127.0.0.1:9090 or [::1]:9090 "+
			"(non-loopback metrics exposure is not supported); got %q", cfg.Metrics.Addr)
	}
	if httpOK && metricsOK && httpPort == metricsPort &&
		(httpHost == metricsHost || httpHost == "" || metricsHost == "") {
		r.fail(EnvMetricsAddr, "must not use the same listener as %s (%q)", EnvHTTPAddr, cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadHeaderTimeout > 0 && cfg.HTTP.ReadTimeout > 0 && cfg.HTTP.ReadHeaderTimeout > cfg.HTTP.ReadTimeout {
		r.fail(EnvHTTPReadHeaderTimeout, "must not exceed %s (%s); got %s",
			EnvHTTPReadTimeout, cfg.HTTP.ReadTimeout, cfg.HTTP.ReadHeaderTimeout)
	}

	if len(r.errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(r.errs...))
	}
	return cfg, nil
}

// reader accumulates validation errors while reading variables.
type reader struct {
	lookup LookupFunc
	errs   []error
}

func (r *reader) fail(key, format string, args ...any) {
	r.errs = append(r.errs, fmt.Errorf("%s: %s", key, fmt.Sprintf(format, args...)))
}

// str returns the trimmed value of key, or def when the variable is unset or blank.
func (r *reader) str(key, def string) string {
	v, ok := r.lookup(key)
	if !ok {
		return def
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	return v
}

func (r *reader) duration(key string, def time.Duration) time.Duration {
	raw := r.str(key, "")
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		r.fail(key, "must be a positive duration such as 5s or 1m30s; got %q", raw)
		return 0
	}
	return d
}

func (r *reader) integer(key string, def, minimum, maximum int) int {
	raw := r.str(key, "")
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < minimum || n > maximum {
		r.fail(key, "must be a whole number from %d to %d; got %q", minimum, maximum, raw)
		return def
	}
	return n
}

// database reads and validates the PostgreSQL settings. The password has no default (AGENTS.md
// §7.7) and is never included in an error message.
func (r *reader) database(env Environment) DatabaseConfig {
	defaultSSLMode := SSLModeDisable
	if env == EnvironmentProduction {
		defaultSSLMode = SSLModeVerifyFull
	}
	db := DatabaseConfig{
		Host:            r.str(EnvDBHost, DefaultDBHost),
		Port:            r.integer(EnvDBPort, DefaultDBPort, 1, 65535),
		Name:            r.str(EnvDBName, DefaultDBName),
		User:            r.str(EnvDBUser, DefaultDBUser),
		SSLMode:         SSLMode(strings.ToLower(r.str(EnvDBSSLMode, string(defaultSSLMode)))),
		MaxConns:        r.integer(EnvDBMaxConns, DefaultDBMaxConns, 1, MaxDBConns),
		MinConns:        r.integer(EnvDBMinConns, DefaultDBMinConns, 0, MaxDBConns),
		ConnectTimeout:  r.duration(EnvDBConnectTimeout, DefaultDBConnectTimeout),
		MaxConnLifetime: r.duration(EnvDBMaxConnLifetime, DefaultDBMaxConnLifetime),
		MaxConnIdleTime: r.duration(EnvDBMaxConnIdleTime, DefaultDBMaxConnIdleTime),
	}

	// The password is read without trimming: leading or trailing spaces may be part of it.
	if pw, ok := r.lookup(EnvDBPassword); ok && pw != "" {
		db.Password = Secret(pw)
	} else {
		r.fail(EnvDBPassword, "is required (secrets have no default)")
	}

	if strings.ContainsAny(db.Host, " \t\r\n,") {
		r.fail(EnvDBHost, "must be a single host name or IP address; got %q", db.Host)
	}
	r.identifier(EnvDBName, db.Name)
	r.identifier(EnvDBUser, db.User)

	switch db.SSLMode {
	case SSLModeDisable, SSLModeRequire, SSLModeVerifyCA, SSLModeVerifyFull:
	default:
		r.fail(EnvDBSSLMode, "must be one of disable, require, verify-ca, verify-full; got %q", db.SSLMode)
	}
	if env == EnvironmentProduction && db.SSLMode == SSLModeDisable {
		r.fail(EnvDBSSLMode, "must not be disable when %s=production (database traffic must be encrypted)", EnvAppEnv)
	}
	if db.MinConns > db.MaxConns {
		r.fail(EnvDBMinConns, "must not exceed %s (%d); got %d", EnvDBMaxConns, db.MaxConns, db.MinConns)
	}
	return db
}

// identifier validates a PostgreSQL database or role name.
func (r *reader) identifier(key, value string) {
	if len(value) > maxIdentifierLength {
		r.fail(key, "must be at most %d bytes; got %d", maxIdentifierLength, len(value))
	}
	for _, c := range value {
		if c < 0x20 || c == 0x7f {
			r.fail(key, "must not contain control characters")
			return
		}
	}
}

func (r *reader) logLevel(key string, def slog.Level) slog.Level {
	raw := strings.ToLower(r.str(key, ""))
	switch raw {
	case "":
		return def
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		r.fail(key, "must be one of debug, info, warn, error; got %q", raw)
		return def
	}
}

// address validates a host:port listen address. The host may be empty (all interfaces),
// "localhost", or an IP literal; the port must be 1-65535.
func (r *reader) address(key, addr string) (host string, port int, ok bool) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		r.fail(key, "must be a host:port listen address such as 127.0.0.1:8080; got %q", addr)
		return "", 0, false
	}
	port, err = strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		r.fail(key, "port must be a number from 1 to 65535; got %q", addr)
		return "", 0, false
	}
	if host != "" && host != "localhost" && net.ParseIP(host) == nil {
		r.fail(key, "host must be empty, localhost, or an IP address; got %q", addr)
		return "", 0, false
	}
	return host, port, true
}

// isLoopbackIP reports whether host is a loopback IP literal (127.0.0.0/8 or ::1).
//
// Host names, including "localhost", are deliberately not accepted: validation does not resolve
// names, and a name could resolve to a non-loopback address at bind time. An empty host means
// "all interfaces" and is therefore not loopback.
func isLoopbackIP(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
