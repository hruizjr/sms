// Package observability provides the process logger and Prometheus metrics.
package observability

import (
	"io"
	"log/slog"

	"github.com/hruizjr/sms/backend/internal/config"
)

// NewLogger returns a structured logger writing to w in the configured format and level. Every
// entry carries service=<service> (for example "api" or "migrate").
//
// Text output is intended for local development; JSON output is intended for every other
// environment. Callers must never pass secrets, message content, or full phone numbers as log
// attributes (AGENTS.md §7.6).
func NewLogger(w io.Writer, cfg config.LogConfig, service string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.Level}

	var handler slog.Handler
	if cfg.Format == config.LogFormatJSON {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	return slog.New(handler).With("service", service)
}
