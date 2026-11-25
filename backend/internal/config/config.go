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
	EnvAppEnv                  = "APP_ENV"
	EnvHTTPAddr                = "HTTP_ADDR"
	EnvHTTPReadTimeout         = "HTTP_READ_TIMEOUT"
	EnvHTTPReadHeaderTimeout   = "HTTP_READ_HEADER_TIMEOUT"
	EnvHTTPWriteTimeout        = "HTTP_WRITE_TIMEOUT"
	EnvHTTPIdleTimeout         = "HTTP_IDLE_TIMEOUT"
	EnvShutdownTimeout         = "SHUTDOWN_TIMEOUT"
	EnvMetricsAddr             = "METRICS_ADDR"
	EnvMetricsAllowNonLoopback = "METRICS_ALLOW_NON_LOOPBACK"
	EnvLogFormat               = "LOG_FORMAT"
	EnvLogLevel                = "LOG_LEVEL"
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
)

// Config is the validated configuration of the API process.
type Config struct {
	Env             Environment
	HTTP            HTTPConfig
	Metrics         MetricsConfig
	Log             LogConfig
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
type MetricsConfig struct {
	Addr string
	// AllowNonLoopback must be set explicitly before the metrics listener may bind to a
	// non-loopback address. It exists so that a container deployment can opt in deliberately;
	// the metrics endpoint must never be exposed publicly.
	AllowNonLoopback bool
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
			Addr:             r.str(EnvMetricsAddr, DefaultMetricsAddr),
			AllowNonLoopback: r.boolean(EnvMetricsAllowNonLoopback, false),
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

	httpHost, httpPort, httpOK := r.address(EnvHTTPAddr, cfg.HTTP.Addr)
	metricsHost, metricsPort, metricsOK := r.address(EnvMetricsAddr, cfg.Metrics.Addr)

	if metricsOK && !cfg.Metrics.AllowNonLoopback && !isLoopbackHost(metricsHost) {
		r.fail(EnvMetricsAddr, "must bind to a loopback address (127.0.0.1, ::1, or localhost) unless %s=true; got %q",
			EnvMetricsAllowNonLoopback, cfg.Metrics.Addr)
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

func (r *reader) boolean(key string, def bool) bool {
	raw := r.str(key, "")
	if raw == "" {
		return def
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		r.fail(key, "must be true or false; got %q", raw)
		return def
	}
	return b
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

// isLoopbackHost reports whether host refers only to the loopback interface.
// An empty host means "all interfaces" and is therefore not loopback.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
