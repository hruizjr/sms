package config

import (
	"strings"
	"time"
)

// Messaging environment variables (Phase 4 Group 2). Documented in backend/README.md and
// backend/.env.example.
const (
	EnvSMSProvider = "SMS_PROVIDER"

	EnvSimulatorReceiptDelay = "SIMULATOR_RECEIPT_DELAY"
	EnvSimulatorMaxSegments  = "SIMULATOR_MAX_SEGMENTS"
	EnvSimulatorReportWindow = "SIMULATOR_REPORT_WINDOW"

	EnvDispatchPollInterval   = "DISPATCH_POLL_INTERVAL"
	EnvDispatchBatchSize      = "DISPATCH_BATCH_SIZE"
	EnvDispatchConcurrency    = "DISPATCH_CONCURRENCY"
	EnvDispatchSubmitTimeout  = "DISPATCH_SUBMIT_TIMEOUT"
	EnvDispatchLease          = "DISPATCH_LEASE"
	EnvDispatchMaxAttempts    = "DISPATCH_MAX_ATTEMPTS"
	EnvDispatchRetryBaseDelay = "DISPATCH_RETRY_BASE_DELAY"
	EnvDispatchRetryMaxDelay  = "DISPATCH_RETRY_MAX_DELAY"
	EnvDispatchMaxMessageAge  = "DISPATCH_MAX_MESSAGE_AGE"
)

// ProviderSimulator is the only provider that exists (simulator-only development, decision log
// §15). Real provider adapters are added under their own approval, with their own settings.
const ProviderSimulator = "simulator"

// Defaults. The simulator values are synthetic. The dispatch values are engineering defaults for
// local development, not telco terms (real limits come from provider agreements, AGENTS.md §11.4).
const (
	DefaultSimulatorReceiptDelay  = time.Second
	DefaultDispatchPollInterval   = 500 * time.Millisecond
	DefaultDispatchBatchSize      = 20
	DefaultDispatchConcurrency    = 4
	DefaultDispatchSubmitTimeout  = 10 * time.Second
	DefaultDispatchLease          = time.Minute
	DefaultDispatchMaxAttempts    = 5
	DefaultDispatchRetryBaseDelay = 2 * time.Second
	DefaultDispatchRetryMaxDelay  = 5 * time.Minute
	DefaultDispatchMaxMessageAge  = 24 * time.Hour
)

// ProviderConfig selects and configures the SMS provider. It is read by the API (for the
// provider's capabilities) and by the dispatcher.
type ProviderConfig struct {
	// Name is the provider adapter: "simulator", or "" when none is configured (production, until a
	// real adapter is approved).
	Name      string
	Simulator SimulatorConfig
}

// SimulatorConfig configures the simulated provider. Every value is synthetic.
type SimulatorConfig struct {
	ReceiptDelay time.Duration
	// MaxSegments is a synthetic segment limit; 0 declares it unknown (not enforced).
	MaxSegments int
	// ReportWindow is the synthetic reporting window after which a submitted message without a
	// final report becomes unknown; 0 disables it (there is no universal default).
	ReportWindow time.Duration
}

// DispatchConfig configures the dispatcher.
type DispatchConfig struct {
	PollInterval   time.Duration
	BatchSize      int
	Concurrency    int
	SubmitTimeout  time.Duration
	Lease          time.Duration
	MaxAttempts    int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
	MaxMessageAge  time.Duration
}

// ReportWindow returns the reporting window of the configured provider.
func (c ProviderConfig) ReportWindow() time.Duration {
	if c.Name == ProviderSimulator {
		return c.Simulator.ReportWindow
	}
	return 0
}

func (r *reader) provider(env Environment) ProviderConfig {
	// The simulator is the default outside production. Production has no default: no production
	// provider exists yet, so Name stays empty and the dispatcher refuses to start.
	def := ProviderSimulator
	if env == EnvironmentProduction {
		def = ""
	}
	cfg := ProviderConfig{Name: strings.ToLower(r.str(EnvSMSProvider, def))}
	switch cfg.Name {
	case "":
	case ProviderSimulator:
		// The simulator never routes to a real network and must never serve production traffic
		// (decision S6, gate PG-6).
		if env == EnvironmentProduction {
			r.fail(EnvSMSProvider, "the simulator cannot be used in production")
		}
	default:
		r.fail(EnvSMSProvider, "must be %q (the only provider available); got %q", ProviderSimulator, cfg.Name)
	}
	cfg.Simulator = SimulatorConfig{
		ReceiptDelay: r.optionalDuration(EnvSimulatorReceiptDelay, DefaultSimulatorReceiptDelay),
		MaxSegments:  r.integer(EnvSimulatorMaxSegments, 0, 0, 1000),
		ReportWindow: r.optionalDuration(EnvSimulatorReportWindow, 0),
	}
	return cfg
}

func (r *reader) dispatch() DispatchConfig {
	cfg := DispatchConfig{
		PollInterval:   r.duration(EnvDispatchPollInterval, DefaultDispatchPollInterval),
		BatchSize:      r.integer(EnvDispatchBatchSize, DefaultDispatchBatchSize, 1, 500),
		Concurrency:    r.integer(EnvDispatchConcurrency, DefaultDispatchConcurrency, 1, 64),
		SubmitTimeout:  r.duration(EnvDispatchSubmitTimeout, DefaultDispatchSubmitTimeout),
		Lease:          r.duration(EnvDispatchLease, DefaultDispatchLease),
		MaxAttempts:    r.integer(EnvDispatchMaxAttempts, DefaultDispatchMaxAttempts, 1, 100),
		RetryBaseDelay: r.duration(EnvDispatchRetryBaseDelay, DefaultDispatchRetryBaseDelay),
		RetryMaxDelay:  r.duration(EnvDispatchRetryMaxDelay, DefaultDispatchRetryMaxDelay),
		MaxMessageAge:  r.duration(EnvDispatchMaxMessageAge, DefaultDispatchMaxMessageAge),
	}
	if cfg.Lease > 0 && cfg.SubmitTimeout > 0 && cfg.Lease <= cfg.SubmitTimeout {
		r.fail(EnvDispatchLease, "must exceed %s (%s); got %s", EnvDispatchSubmitTimeout, cfg.SubmitTimeout, cfg.Lease)
	}
	if cfg.RetryBaseDelay > 0 && cfg.RetryMaxDelay > 0 && cfg.RetryBaseDelay > cfg.RetryMaxDelay {
		r.fail(EnvDispatchRetryMaxDelay, "must not be less than %s (%s); got %s", EnvDispatchRetryBaseDelay, cfg.RetryBaseDelay, cfg.RetryMaxDelay)
	}
	return cfg
}

// optionalDuration is like duration but accepts "0" (meaning "disabled") as well.
func (r *reader) optionalDuration(key string, def time.Duration) time.Duration {
	raw := r.str(key, "")
	if raw == "" {
		return def
	}
	if raw == "0" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		r.fail(key, "must be a duration such as 2s or 5m, or 0; got %q", raw)
		return def
	}
	return d
}
