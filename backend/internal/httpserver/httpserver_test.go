package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hruizjr/sms/backend/internal/observability"
)

// safeBuffer is a bytes.Buffer safe for concurrent use. Server tests log from several
// goroutines (listeners and request handlers) while the test reads the output.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func testLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// fakePinger is a hand-written Pinger for unit tests. ping, when set, decides the result.
type fakePinger struct {
	calls atomic.Int32
	ping  func(ctx context.Context) error
}

func (f *fakePinger) Ping(ctx context.Context) error {
	f.calls.Add(1)
	if f.ping == nil {
		return nil
	}
	return f.ping(ctx)
}

func testDeps(ready bool) (RouterDeps, *safeBuffer) {
	var buf safeBuffer
	readiness := &Readiness{}
	readiness.SetReady(ready)
	return RouterDeps{
		Logger:       testLogger(&buf),
		Metrics:      observability.NewMetrics(),
		Readiness:    readiness,
		Database:     &fakePinger{},
		NewRequestID: func() string { return "req-test" },
	}, &buf
}

// do serves one request in-process and returns the recorder and body.
func do(t *testing.T, h http.Handler, method, target string) (*httptest.ResponseRecorder, []byte) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), method, target, nil))
	return rec, rec.Body.Bytes()
}

func decodeProblem(t *testing.T, res *httptest.ResponseRecorder, body []byte) problem {
	t.Helper()
	if ct := res.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	var p problem
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("body is not a problem document: %v (%s)", err, body)
	}
	return p
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	for _, ready := range []bool{true, false} {
		deps, _ := testDeps(ready)
		res, body := do(t, NewRouter(deps), http.MethodGet, "/healthz")

		if res.Code != http.StatusOK {
			t.Errorf("ready=%v: status = %d, want 200 (liveness must not depend on readiness)", ready, res.Code)
		}
		if ct := res.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if strings.TrimSpace(string(body)) != `{"status":"ok"}` {
			t.Errorf("body = %s, want {\"status\":\"ok\"}", body)
		}
	}
}

func TestReadyz(t *testing.T) {
	t.Parallel()

	t.Run("ready", func(t *testing.T) {
		t.Parallel()
		deps, _ := testDeps(true)
		res, body := do(t, NewRouter(deps), http.MethodGet, "/readyz")
		if res.Code != http.StatusOK || strings.TrimSpace(string(body)) != `{"status":"ready"}` {
			t.Errorf("got %d %s, want 200 {\"status\":\"ready\"}", res.Code, body)
		}
	})

	t.Run("not ready", func(t *testing.T) {
		t.Parallel()
		deps, _ := testDeps(false)
		res, body := do(t, NewRouter(deps), http.MethodGet, "/readyz")
		if res.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", res.Code)
		}
		p := decodeProblem(t, res, body)
		if p.Status != http.StatusServiceUnavailable || p.Title != "Service Unavailable" || p.RequestID != "req-test" {
			t.Errorf("problem = %+v", p)
		}
	})
}

func TestStandardResponseHeaders(t *testing.T) {
	t.Parallel()

	deps, _ := testDeps(true)
	res, _ := do(t, NewRouter(deps), http.MethodGet, "/healthz")
	for k, v := range map[string]string{
		"X-Request-ID":           "req-test",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "no-store",
	} {
		if got := res.Header().Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
}

func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method, target string
		want           int
	}{
		{http.MethodGet, "/does-not-exist", http.StatusNotFound},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			t.Parallel()
			deps, _ := testDeps(true)
			res, body := do(t, NewRouter(deps), tt.method, tt.target)
			if res.Code != tt.want {
				t.Fatalf("status = %d, want %d", res.Code, tt.want)
			}
			p := decodeProblem(t, res, body)
			if p.Type != "about:blank" || p.Status != tt.want || p.Title != http.StatusText(tt.want) {
				t.Errorf("problem = %+v", p)
			}
		})
	}
}

func TestPanicProducesGenericProblem(t *testing.T) {
	t.Parallel()

	deps, logs := testDeps(true)
	h := newRouter(deps, func(r chi.Router) {
		r.Get("/boom", func(http.ResponseWriter, *http.Request) {
			panic("synthetic internal detail: connection string user=app password=synthetic-pa55word")
		})
	})

	res, body := do(t, h, http.MethodGet, "/boom")
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.Code)
	}
	p := decodeProblem(t, res, body)
	if p.Detail != "An unexpected error occurred." || p.RequestID != "req-test" {
		t.Errorf("problem = %+v", p)
	}
	if strings.Contains(string(body), "synthetic internal detail") || strings.Contains(string(body), "synthetic-pa55word") ||
		strings.Contains(string(body), "goroutine") {
		t.Errorf("internal details leaked to the client: %s", body)
	}
	log := logs.String()
	if !strings.Contains(log, `"msg":"panic recovered"`) || !strings.Contains(log, `"panic_type":"string"`) {
		t.Errorf("expected a panic log recording the panic type, got: %s", log)
	}
	if !strings.Contains(log, `"msg":"http request"`) || !strings.Contains(log, `"status":500`) {
		t.Errorf("expected a 500 access log entry, got: %s", log)
	}
	for _, leak := range []string{"synthetic internal detail", "synthetic-pa55word", "user=app"} {
		if strings.Contains(log, leak) {
			t.Errorf("panic value leaked to the log (%q): %s", leak, log)
		}
	}
}

// startServer runs Serve on ephemeral loopback ports and returns the base URLs.
func startServer(t *testing.T, app http.Handler, shutdownTimeout time.Duration) (appURL, metricsURL string, readiness *Readiness, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	var buf safeBuffer
	logger := testLogger(&buf) // one logger shared by router and server, as in production
	metrics := observability.NewMetrics()
	readiness = &Readiness{}
	if app == nil {
		app = NewRouter(RouterDeps{Logger: logger, Metrics: metrics, Readiness: readiness, Database: &fakePinger{}})
	}
	srv := NewServer(ServerConfig{
		ReadTimeout: 5 * time.Second, ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second, IdleTimeout: 5 * time.Second,
		ShutdownTimeout: shutdownTimeout,
	}, logger, app, metrics.Handler(), readiness)

	var lc net.ListenConfig
	appLn, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	metricsLn, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		errCh <- srv.Serve(ctx, appLn, metricsLn)
		close(finished)
	}()
	// Tests may consume errCh themselves, so cleanup waits on finished instead.
	t.Cleanup(func() {
		cancelFn()
		<-finished
	})
	return "http://" + appLn.Addr().String(), "http://" + metricsLn.Addr().String(), readiness, cancelFn, errCh
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, err := io.ReadAll(res.Body)
	if closeErr := res.Body.Close(); closeErr != nil {
		t.Fatalf("close body: %v", closeErr)
	}
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return res.StatusCode, string(body)
}

func TestServerStartupAndMetricsListener(t *testing.T) {
	t.Parallel()

	appURL, metricsURL, readiness, _, _ := startServer(t, nil, 5*time.Second)

	if code, body := get(t, appURL+"/healthz"); code != http.StatusOK || !strings.Contains(body, `"ok"`) {
		t.Errorf("/healthz = %d %s", code, body)
	}
	if !readiness.Ready() {
		t.Error("server should be ready while serving")
	}
	if code, _ := get(t, appURL+"/readyz"); code != http.StatusOK {
		t.Errorf("/readyz = %d, want 200", code)
	}

	code, body := get(t, metricsURL+"/metrics")
	if code != http.StatusOK {
		t.Fatalf("/metrics = %d, want 200", code)
	}
	if !strings.Contains(body, `sms_http_requests_total{code="200",method="GET",route="/healthz"} 1`) {
		t.Errorf("metrics do not record the /healthz request by route pattern:\n%s", body)
	}

	// The metrics endpoint is not served on the application listener, and the metrics listener
	// serves nothing but /metrics.
	if code, _ := get(t, appURL+"/metrics"); code != http.StatusNotFound {
		t.Errorf("application listener /metrics = %d, want 404", code)
	}
	if code, _ := get(t, metricsURL+"/healthz"); code != http.StatusNotFound {
		t.Errorf("metrics listener /healthz = %d, want 404", code)
	}
}

func TestServerGracefulShutdownWaitsForInFlightRequests(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})
	appURL, _, readiness, cancel, done := startServer(t, app, 5*time.Second)

	type response struct {
		code int
		err  error
	}
	respCh := make(chan response, 1)
	go func() {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, appURL+"/slow", nil)
		if err != nil {
			respCh <- response{err: err}
			return
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			respCh <- response{err: err}
			return
		}
		_ = res.Body.Close()
		respCh <- response{code: res.StatusCode}
	}()

	<-started
	cancel()

	// While the request is in flight, Serve must not have returned.
	select {
	case err := <-done:
		t.Fatalf("Serve returned (%v) while a request was still in flight", err)
	default:
	}

	close(release)
	if r := <-respCh; r.err != nil || r.code != http.StatusOK {
		t.Fatalf("in-flight request = %+v, want 200 without error", r)
	}
	if err := <-done; err != nil {
		t.Fatalf("Serve returned %v, want nil after graceful shutdown", err)
	}
	if readiness.Ready() {
		t.Error("server must not report ready after shutdown")
	}
}

func TestServerShutdownTimeout(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})
	appURL, _, _, cancel, done := startServer(t, app, 50*time.Millisecond)

	go func() {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, appURL+"/stuck", nil)
		if err != nil {
			return
		}
		if res, err := http.DefaultClient.Do(req); err == nil {
			_ = res.Body.Close()
		}
	}()
	<-started
	cancel()

	err := <-done
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Serve returned %v, want an error wrapping context.DeadlineExceeded", err)
	}
}

func TestRunReportsBindFailure(t *testing.T) {
	t.Parallel()

	var lc net.ListenConfig
	occupied, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	var buf safeBuffer
	srv := NewServer(ServerConfig{
		Addr:            occupied.Addr().String(),
		MetricsAddr:     "127.0.0.1:0",
		ShutdownTimeout: time.Second,
	}, testLogger(&buf), http.NotFoundHandler(), http.NotFoundHandler(), &Readiness{})

	if err := srv.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "listen on application address") {
		t.Fatalf("Run() = %v, want an application listen error", err)
	}
}

func TestRunStopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	var buf safeBuffer
	srv := NewServer(ServerConfig{
		Addr:            "127.0.0.1:0",
		MetricsAddr:     "127.0.0.1:0",
		ShutdownTimeout: time.Second,
	}, testLogger(&buf), http.NotFoundHandler(), http.NotFoundHandler(), &Readiness{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := srv.Run(ctx); err != nil {
		t.Fatalf("Run() with a cancelled context = %v, want nil", err)
	}
	if !strings.Contains(buf.String(), "api stopped") {
		t.Errorf("expected a clean stop log, got %s", buf.String())
	}
}
