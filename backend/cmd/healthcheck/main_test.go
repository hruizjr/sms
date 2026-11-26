package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTripFunc is an in-memory transport: no sockets are opened.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func respond(status int, header http.Header) roundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		if header == nil {
			header = http.Header{}
		}
		return &http.Response{
			StatusCode: status,
			Header:     header,
			Body:       io.NopCloser(strings.NewReader(`{"status":"ready"}`)),
			Request:    r,
		}, nil
	}
}

func TestRunProbesReadinessEndpoint(t *testing.T) {
	t.Parallel()

	var gotMethod, gotURL string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotMethod, gotURL = r.Method, r.URL.String()
		return respond(http.StatusOK, nil)(r)
	})
	var stderr bytes.Buffer
	if code := run(nil, transport, &stderr); code != exitHealthy {
		t.Fatalf("exit code %d, want %d; stderr %q", code, exitHealthy, stderr.String())
	}
	if gotMethod != http.MethodGet || gotURL != "http://127.0.0.1:8080/readyz" {
		t.Fatalf("request %s %s, want GET http://127.0.0.1:8080/readyz", gotMethod, gotURL)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr %q", stderr.String())
	}
}

func TestRunUnhealthy(t *testing.T) {
	t.Parallel()

	blockUntilCancelled := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	tests := []struct {
		name      string
		args      []string
		transport http.RoundTripper
		wantErr   string
	}{
		{"not ready", nil, respond(http.StatusServiceUnavailable, nil), "answered 503"},
		{"server error", nil, respond(http.StatusInternalServerError, nil), "answered 500"},
		{
			"redirect is not followed", nil,
			respond(http.StatusFound, http.Header{"Location": []string{"http://127.0.0.1:8080/elsewhere"}}),
			"answered 302",
		},
		{
			"connection refused", nil,
			roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("connection refused") }),
			"connection refused",
		},
		{"timeout", []string{"-timeout", "20ms"}, blockUntilCancelled, "context deadline exceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			start := time.Now()
			if code := run(tt.args, tt.transport, &stderr); code != exitUnhealthy {
				t.Fatalf("exit code %d, want %d", code, exitUnhealthy)
			}
			if !strings.Contains(stderr.String(), tt.wantErr) {
				t.Fatalf("stderr %q does not contain %q", stderr.String(), tt.wantErr)
			}
			if time.Since(start) > 2*time.Second {
				t.Fatalf("probe took %s; the timeout was not applied", time.Since(start))
			}
		})
	}
}

func TestRunUsageErrors(t *testing.T) {
	t.Parallel()

	neverCalled := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport must not be called on a usage error")
	})
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"https", []string{"-url", "https://127.0.0.1:8080/readyz"}, "scheme must be http"},
		{"host name", []string{"-url", "http://localhost:8080/readyz"}, "loopback IP address"},
		{"non-loopback IP", []string{"-url", "http://10.0.0.5:8080/readyz"}, "loopback IP address"},
		{"no port", []string{"-url", "http://127.0.0.1/readyz"}, "loopback IP address"},
		{"unparseable", []string{"-url", "http://[::1"}, "invalid -url"},
		{"extra argument", []string{"now"}, `unexpected argument "now"`},
		{"zero timeout", []string{"-timeout", "0s"}, "-timeout must be positive"},
		{"unknown flag", []string{"-verbose"}, "flag provided but not defined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			if code := run(tt.args, neverCalled, &stderr); code != exitUsage {
				t.Fatalf("exit code %d, want %d; stderr %q", code, exitUsage, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantErr) {
				t.Fatalf("stderr %q does not contain %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestRunAcceptsIPv6Loopback(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	if code := run([]string{"-url", "http://[::1]:8080/readyz"}, respond(http.StatusOK, nil), &stderr); code != exitHealthy {
		t.Fatalf("exit code %d, want %d; stderr %q", code, exitHealthy, stderr.String())
	}
}
