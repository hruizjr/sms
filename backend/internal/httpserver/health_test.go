package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// errSyntheticDB stands in for a driver error that names internal details. None of its text may
// reach a client.
var errSyntheticDB = errors.New("failed to connect to `user=synthetic_user database=synthetic_db`: 10.9.8.7:5432: connection refused")

func TestReadyzChecksDatabase(t *testing.T) {
	t.Parallel()

	deps, _ := testDeps(true)
	db := &fakePinger{}
	deps.Database = db

	res, body := do(t, NewRouter(deps), http.MethodGet, "/readyz")
	if res.Code != http.StatusOK || strings.TrimSpace(string(body)) != `{"status":"ready"}` {
		t.Errorf("got %d %s, want 200 {\"status\":\"ready\"}", res.Code, body)
	}
	if db.calls.Load() != 1 {
		t.Errorf("database pinged %d times, want 1", db.calls.Load())
	}
}

func TestReadyzDatabaseUnavailable(t *testing.T) {
	t.Parallel()

	deps, logs := testDeps(true)
	deps.Database = &fakePinger{ping: func(context.Context) error { return errSyntheticDB }}

	res, body := do(t, NewRouter(deps), http.MethodGet, "/readyz")
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.Code)
	}
	p := decodeProblem(t, res, body)
	if p.Detail != notReadyDetail || p.Title != "Service Unavailable" || p.RequestID != "req-test" {
		t.Errorf("problem = %+v", p)
	}
	for _, leak := range []string{"connect", "synthetic_user", "synthetic_db", "10.9.8.7", "5432", "database"} {
		if strings.Contains(string(body), leak) {
			t.Errorf("response reveals internal detail %q: %s", leak, body)
		}
	}
	if !strings.Contains(logs.String(), `"msg":"readiness check failed"`) || !strings.Contains(logs.String(), "connection refused") {
		t.Errorf("the failure should be logged server-side with its cause: %s", logs.String())
	}
}

func TestReadyzLogsDatabaseTransitionsOnce(t *testing.T) {
	t.Parallel()

	deps, logs := testDeps(true)
	var failing atomic.Bool
	deps.Database = &fakePinger{ping: func(context.Context) error {
		if failing.Load() {
			return errSyntheticDB
		}
		return nil
	}}
	h := NewRouter(deps)

	failing.Store(true)
	for range 3 {
		do(t, h, http.MethodGet, "/readyz")
	}
	failing.Store(false)
	for range 2 {
		do(t, h, http.MethodGet, "/readyz")
	}

	out := logs.String()
	if n := strings.Count(out, `"msg":"readiness check failed"`); n != 1 {
		t.Errorf("failure logged %d times, want once per transition: %s", n, out)
	}
	if n := strings.Count(out, `"msg":"readiness check recovered"`); n != 1 {
		t.Errorf("recovery logged %d times, want once: %s", n, out)
	}
}

func TestReadyzPingIsBounded(t *testing.T) {
	t.Parallel()

	deps, _ := testDeps(true)
	deps.ReadinessTimeout = 50 * time.Millisecond
	deps.Database = &fakePinger{ping: func(ctx context.Context) error {
		<-ctx.Done() // a database that never answers
		return ctx.Err()
	}}

	start := time.Now()
	res, _ := do(t, NewRouter(deps), http.MethodGet, "/readyz")
	elapsed := time.Since(start)

	if res.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", res.Code)
	}
	if elapsed > 2*time.Second {
		t.Errorf("readiness took %s; the ping must be bounded by ReadinessTimeout", elapsed)
	}
}

func TestReadyzSkipsDatabaseWhileNotServing(t *testing.T) {
	t.Parallel()

	deps, _ := testDeps(false)
	db := &fakePinger{}
	deps.Database = db

	res, _ := do(t, NewRouter(deps), http.MethodGet, "/readyz")
	if res.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", res.Code)
	}
	if db.calls.Load() != 0 {
		t.Errorf("database pinged %d times while not serving, want 0", db.calls.Load())
	}
}

func TestHealthzIgnoresDatabase(t *testing.T) {
	t.Parallel()

	deps, _ := testDeps(true)
	db := &fakePinger{ping: func(context.Context) error { return errSyntheticDB }}
	deps.Database = db

	res, _ := do(t, NewRouter(deps), http.MethodGet, "/healthz")
	if res.Code != http.StatusOK {
		t.Errorf("/healthz = %d, want 200 even when the database is down", res.Code)
	}
	if db.calls.Load() != 0 {
		t.Errorf("/healthz pinged the database %d times, want 0", db.calls.Load())
	}
}

func TestNewRouterRequiresDatabase(t *testing.T) {
	t.Parallel()

	deps, _ := testDeps(true)
	deps.Database = nil
	defer func() {
		if recover() == nil {
			t.Error("NewRouter without a Database should panic at startup")
		}
	}()
	NewRouter(deps)
}
