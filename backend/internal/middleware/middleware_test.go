package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func jsonLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func fixedID(id string) IDGenerator { return func() string { return id } }

func TestRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		incoming string
		want     string
	}{
		{"generated when missing", "", "generated-id"},
		{"client value reused when valid", "client-ID_1.2", "client-ID_1.2"},
		{"rejected when it contains spaces", "abc def", "generated-id"},
		{"rejected when it contains control characters", "abc\nINFO forged", "generated-id"},
		{"rejected when too long", strings.Repeat("a", maxRequestIDLength+1), "generated-id"},
		{"accepted at maximum length", strings.Repeat("a", maxRequestIDLength), strings.Repeat("a", maxRequestIDLength)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var seen string
			h := RequestID(fixedID("generated-id"))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				seen = RequestIDFromContext(r.Context())
			}))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			if tt.incoming != "" {
				req.Header.Set(RequestIDHeader, tt.incoming)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if seen != tt.want {
				t.Errorf("context request ID = %q, want %q", seen, tt.want)
			}
			if got := rec.Header().Get(RequestIDHeader); got != tt.want {
				t.Errorf("response %s = %q, want %q", RequestIDHeader, got, tt.want)
			}
		})
	}
}

func TestNewRandomID(t *testing.T) {
	t.Parallel()

	a, b := NewRandomID(), NewRandomID()
	if len(a) != 32 || !validRequestID(a) {
		t.Errorf("NewRandomID() = %q, want 32 valid hex characters", a)
	}
	if a == b {
		t.Error("NewRandomID() returned the same value twice")
	}
}

func TestRequestIDFromContextWithoutValue(t *testing.T) {
	t.Parallel()
	if got := RequestIDFromContext(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil).Context()); got != "" {
		t.Errorf("RequestIDFromContext() = %q, want empty", got)
	}
}

func TestStatusRecorder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    int
	}{
		{"nothing written defaults to 200", func(http.ResponseWriter, *http.Request) {}, http.StatusOK},
		{"body only implies 200", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }, http.StatusOK},
		{"explicit status", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }, http.StatusTeapot},
		{"first status wins", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.WriteHeader(http.StatusInternalServerError)
		}, http.StatusAccepted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			underlying := httptest.NewRecorder()
			rec := recordStatus(underlying)
			tt.handler(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

			if rec.Status() != tt.want {
				t.Errorf("Status() = %d, want %d", rec.Status(), tt.want)
			}
			if underlying.Code != tt.want {
				t.Errorf("underlying status = %d, want %d", underlying.Code, tt.want)
			}
			if rec.Unwrap() != underlying {
				t.Error("Unwrap() does not return the underlying writer")
			}
			if recordStatus(rec) != rec {
				t.Error("recordStatus does not reuse an existing recorder")
			}
		})
	}
}

// newTestRouter mounts the middleware under test on a chi router with a parameterised route,
// so that route patterns are populated exactly as in production.
func newTestRouter(mw func(http.Handler) http.Handler, handler http.HandlerFunc) http.Handler {
	r := chi.NewRouter()
	r.Use(RequestID(fixedID("req-123")), mw)
	r.Get("/items/{id}", handler)
	return r
}

func TestAccessLogFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	h := newTestRouter(AccessLog(jsonLogger(&buf)), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	// Synthetic sensitive values: none of them may appear in the log.
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/item-42?msisdn=0000000000&api_key=synthetic-key", strings.NewReader("synthetic-body"))
	req.Header.Set("Authorization", "Bearer synthetic-token")
	h.ServeHTTP(httptest.NewRecorder(), req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v (%q)", err, buf.String())
	}
	want := map[string]any{
		"level":      "INFO",
		"msg":        "http request",
		"request_id": "req-123",
		"method":     "GET",
		"route":      "/items/{id}",
		"status":     float64(http.StatusCreated),
	}
	for k, v := range want {
		if entry[k] != v {
			t.Errorf("log field %q = %v, want %v", k, entry[k], v)
		}
	}
	if _, ok := entry["duration_ms"].(float64); !ok {
		t.Errorf("log field duration_ms missing or not a number: %v", entry["duration_ms"])
	}
	for _, secret := range []string{"item-42", "msisdn", "0000000000", "synthetic-key", "synthetic-token", "synthetic-body", "Authorization"} {
		if strings.Contains(buf.String(), secret) {
			t.Errorf("access log leaked %q: %s", secret, buf.String())
		}
	}
}

func TestAccessLogLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status    int
		wantLevel string
	}{
		{http.StatusOK, "INFO"},
		{http.StatusNotFound, "INFO"},
		{http.StatusInternalServerError, "ERROR"},
		{http.StatusServiceUnavailable, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			h := newTestRouter(AccessLog(jsonLogger(&buf)), func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/1", nil))
			if !strings.Contains(buf.String(), `"level":"`+tt.wantLevel+`"`) {
				t.Errorf("status %d logged as %s, want %s", tt.status, buf.String(), tt.wantLevel)
			}
		})
	}
}

func TestAccessLogUnmatchedRoute(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	h := newTestRouter(AccessLog(jsonLogger(&buf)), func(http.ResponseWriter, *http.Request) {})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/no/such/path", nil))

	if !strings.Contains(buf.String(), `"route":"unmatched"`) || strings.Contains(buf.String(), "/no/such/path") {
		t.Errorf("unmatched request should log route=unmatched without the raw path: %s", buf.String())
	}
}

type recordingObserver struct {
	mu    sync.Mutex
	calls []string
}

func (o *recordingObserver) ObserveHTTPRequest(method, route string, status int, _ time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls = append(o.calls, method+" "+route+" "+http.StatusText(status))
}

func TestMetricsMiddlewareUsesRoutePattern(t *testing.T) {
	t.Parallel()

	obs := &recordingObserver{}
	h := newTestRouter(Metrics(obs), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/secret-value", nil))

	want := []string{"GET /items/{id} Accepted"}
	if len(obs.calls) != 1 || obs.calls[0] != want[0] {
		t.Errorf("observer calls = %v, want %v", obs.calls, want)
	}
}

// syntheticSecret stands in for sensitive data (a connection string with a password) that a
// careless panic might carry. It is not a real credential.
//
//nolint:gosec // G101: deliberate synthetic credential used to prove it is never logged.
const syntheticSecret = "postgres://app:synthetic-pa55word@db.invalid/sms"

// secretCarrier is a non-error, non-string panic value with a sensitive field.
type secretCarrier struct{ DSN string }

func TestRecoverDoesNotLeakPanicValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    any
		wantType string
	}{
		{"string", "failed to connect: " + syntheticSecret, "string"},
		{"error", fmt.Errorf("open database %s: %w", syntheticSecret, errors.New("refused")), "*fmt.wrapError"},
		{"struct", secretCarrier{DSN: syntheticSecret}, "middleware.secretCarrier"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			fallback := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("generic error"))
			})
			h := newTestRouter(Recover(jsonLogger(&buf), fallback), func(http.ResponseWriter, *http.Request) {
				panic(tt.value)
			})

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/1", nil))

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500", rec.Code)
			}
			body, _ := io.ReadAll(rec.Body)
			log := buf.String()
			for _, leak := range []string{syntheticSecret, "synthetic-pa55word", "failed to connect", "refused"} {
				if strings.Contains(string(body), leak) {
					t.Errorf("panic value leaked to the client (%q): %q", leak, body)
				}
				if strings.Contains(log, leak) {
					t.Errorf("panic value leaked to the log (%q): %s", leak, log)
				}
			}

			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("panic log is not a single JSON entry: %v: %s", err, log)
			}
			if entry["msg"] != "panic recovered" || entry["request_id"] != "req-123" || entry["route"] != "/items/{id}" {
				t.Errorf("panic log has wrong fields: %s", log)
			}
			if entry["panic_type"] != tt.wantType {
				t.Errorf("panic_type = %v, want %q", entry["panic_type"], tt.wantType)
			}
			if _, ok := entry["panic"]; ok {
				t.Errorf("panic log must not contain the raw panic value: %s", log)
			}
			stack, _ := entry["stack"].(string)
			if !strings.Contains(stack, "middleware.TestRecoverDoesNotLeakPanicValue") {
				t.Errorf("stack should name the panicking function, got: %s", stack)
			}
		})
	}
}

func TestPanicStackHasNoArgumentValues(t *testing.T) {
	t.Parallel()

	// panicStack must list only function names and file:line positions. runtime/debug.Stack
	// would print each frame's raw argument words, e.g. "stackFromFrame(0x4d2)", and PC offsets
	// such as "+0x1f"; neither may appear.
	stack := stackFromFrame(0x4d2) // 1234, a stand-in for a sensitive integer argument
	if strings.Contains(stack, "0x") {
		t.Errorf("stack contains raw words: %s", stack)
	}
	if !strings.Contains(stack, "middleware.stackFromFrame") || !strings.Contains(stack, "middleware_test.go:") {
		t.Errorf("stack missing function or file:line: %s", stack)
	}
}

//go:noinline
func stackFromFrame(sensitive int) string {
	if sensitive < 0 {
		return ""
	}
	return panicStack()
}

func TestRecoverAfterResponseStarted(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	fallbackCalled := false
	fallback := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { fallbackCalled = true })

	inner := Recover(jsonLogger(&buf), fallback)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		panic("late failure: " + syntheticSecret)
	}))
	rec := httptest.NewRecorder()
	inner.ServeHTTP(recordStatus(rec), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	if !strings.Contains(buf.String(), `"panic_type":"string"`) || strings.Contains(buf.String(), syntheticSecret) {
		t.Errorf("late panic should be logged by type only: %s", buf.String())
	}

	if fallbackCalled {
		t.Error("fallback must not run once the response has started")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the already-sent 200", rec.Code)
	}
}

func TestRecoverRepanicsErrAbortHandler(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	h := Recover(jsonLogger(&buf), http.NotFoundHandler())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		p := recover()
		err, ok := p.(error)
		if !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Errorf("recovered %v, want http.ErrAbortHandler to be re-panicked", p)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	SecurityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-store",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
}

func TestLimitRequestBody(t *testing.T) {
	t.Parallel()

	var readErr error
	h := LimitRequestBody(8)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader("more than eight bytes")))

	var maxErr *http.MaxBytesError
	if !errors.As(readErr, &maxErr) {
		t.Errorf("reading past the limit returned %v, want *http.MaxBytesError", readErr)
	}
}
