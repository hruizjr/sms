package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/hruizjr/sms/backend/internal/config"
)

func TestNewLoggerJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogConfig{Format: config.LogFormatJSON, Level: slog.LevelInfo}, "migrate")
	logger.Info("hello", "request_id", "abc")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON logger output is not JSON: %v (%q)", err, buf.String())
	}
	for key, want := range map[string]any{"msg": "hello", "level": "INFO", "service": "migrate", "request_id": "abc"} {
		if entry[key] != want {
			t.Errorf("entry[%q] = %v, want %v", key, entry[key], want)
		}
	}
}

func TestNewLoggerText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogConfig{Format: config.LogFormatText, Level: slog.LevelInfo}, "api")
	logger.Info("hello")

	out := buf.String()
	if !strings.Contains(out, "msg=hello") || !strings.Contains(out, "service=api") {
		t.Errorf("text logger output = %q, want key=value pairs", out)
	}
}

func TestNewLoggerLevelFiltering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogConfig{Format: config.LogFormatJSON, Level: slog.LevelWarn}, "api")
	logger.Info("dropped")
	logger.Debug("dropped")
	logger.Warn("kept")

	out := buf.String()
	if strings.Contains(out, "dropped") || !strings.Contains(out, "kept") {
		t.Errorf("level filtering failed, output = %q", out)
	}
}

func TestObserveHTTPRequestLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		method, route         string
		status                int
		wantMethod, wantRoute string
		wantCode              string
	}{
		{"matched route", http.MethodGet, "/healthz", 200, "GET", "/healthz", "200"},
		{"unmatched route", http.MethodGet, "", 404, "GET", UnmatchedRoute, "404"},
		{"non-standard method", "BREW", "/healthz", 405, otherMethod, "/healthz", "405"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewMetrics()
			m.ObserveHTTPRequest(tt.method, tt.route, tt.status, 10*time.Millisecond)

			got := testutil.ToFloat64(m.requests.WithLabelValues(tt.wantMethod, tt.wantRoute, tt.wantCode))
			if got != 1 {
				t.Errorf("requests_total{%s,%s,%s} = %v, want 1", tt.wantMethod, tt.wantRoute, tt.wantCode, got)
			}
			if n := testutil.CollectAndCount(m.duration); n != 1 {
				t.Errorf("duration histogram series = %d, want 1", n)
			}
		})
	}
}

func TestMetricsHandlerExposesHTTPMetrics(t *testing.T) {
	t.Parallel()

	m := NewMetrics()
	m.ObserveHTTPRequest(http.MethodGet, "/healthz", 200, time.Millisecond)

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil))

	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	for _, name := range []string{"sms_http_requests_total", "sms_http_request_duration_seconds", "go_goroutines"} {
		if !strings.Contains(string(body), name) {
			t.Errorf("metrics output does not contain %s", name)
		}
	}
}

// Sign-in limit cleanup cycles: successes and failures counted separately, a failure never moves the
// last-success time, incomplete cycles counted, deletions per table; every series exists before the
// first cycle.
func TestObserveSignInLimitCleanup(t *testing.T) {
	t.Parallel()

	m := NewMetrics()
	if n := testutil.CollectAndCount(m.cleanupRuns); n != 2 {
		t.Fatalf("%d cleanup run series before any cycle, want 2 (success, failure)", n)
	}
	if got := testutil.ToFloat64(m.cleanupLastSuccess); got != 0 {
		t.Fatalf("last success before any cycle = %v, want 0", got)
	}

	before := float64(time.Now().Unix())
	m.ObserveSignInLimitCleanup(true, true, 20*time.Millisecond, 7, 3)
	last := testutil.ToFloat64(m.cleanupLastSuccess)
	if last < before {
		t.Errorf("last success = %v, want at least %v", last, before)
	}
	m.ObserveSignInLimitCleanup(false, false, time.Second, 2, 0)
	m.ObserveSignInLimitCleanup(true, false, time.Second, 50000, 0)

	for _, tt := range []struct {
		name string
		got  float64
		want float64
	}{
		{"success runs", testutil.ToFloat64(m.cleanupRuns.WithLabelValues("success")), 2},
		{"failed runs", testutil.ToFloat64(m.cleanupRuns.WithLabelValues("failure")), 1},
		{"incomplete runs", testutil.ToFloat64(m.cleanupIncomplete), 2},
		{"budgets deleted", testutil.ToFloat64(m.cleanupDeleted.WithLabelValues("failure_budgets")), 50009},
		{"buckets deleted", testutil.ToFloat64(m.cleanupDeleted.WithLabelValues("request_buckets")), 3},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
	if got := testutil.ToFloat64(m.cleanupLastSuccess); got < last {
		t.Errorf("last success moved backwards: %v < %v", got, last)
	}

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	for _, name := range []string{
		"sms_signin_limit_cleanup_runs_total", "sms_signin_limit_cleanup_incomplete_runs_total",
		"sms_signin_limit_cleanup_duration_seconds", "sms_signin_limit_cleanup_deleted_rows_total",
		"sms_signin_limit_cleanup_last_success_timestamp_seconds",
	} {
		if !strings.Contains(body, name) {
			t.Errorf("metrics output does not contain %s", name)
		}
	}
}
