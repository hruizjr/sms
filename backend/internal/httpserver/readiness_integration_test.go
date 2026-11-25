//go:build integration

package httpserver_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/httpserver"
	"github.com/hruizjr/sms/backend/internal/observability"
)

func router(t *testing.T, cfg config.DatabaseConfig) http.Handler {
	t.Helper()
	db, err := database.Open(context.Background(), cfg, "sms-readiness-test")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(db.Close)

	readiness := &httpserver.Readiness{}
	readiness.SetReady(true) // as the server does once its listeners are serving
	var logs bytes.Buffer
	return httpserver.NewRouter(httpserver.RouterDeps{
		Logger:    slog.New(slog.NewJSONHandler(&logs, nil)),
		Metrics:   observability.NewMetrics(),
		Readiness: readiness,
		Database:  db,
	})
}

func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	body, _ := io.ReadAll(rec.Body)
	return rec.Code, string(body)
}

func TestIntegrationReadyzWithDatabaseAvailable(t *testing.T) {
	t.Parallel()
	h := router(t, dbtest.NewDatabase(t))

	if code, body := get(t, h, "/readyz"); code != http.StatusOK || strings.TrimSpace(body) != `{"status":"ready"}` {
		t.Errorf("/readyz = %d %s, want 200 {\"status\":\"ready\"}", code, body)
	}
	if code, _ := get(t, h, "/healthz"); code != http.StatusOK {
		t.Errorf("/healthz = %d, want 200", code)
	}
}

func TestIntegrationReadyzWithDatabaseUnavailable(t *testing.T) {
	t.Parallel()

	cases := map[string]func(t *testing.T, cfg *config.DatabaseConfig){
		"server unreachable": func(t *testing.T, cfg *config.DatabaseConfig) {
			ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			cfg.Host = "127.0.0.1"
			cfg.Port = ln.Addr().(*net.TCPAddr).Port
			_ = ln.Close() // nothing listens on the port any more
		},
		"wrong password": func(_ *testing.T, cfg *config.DatabaseConfig) {
			cfg.Password = config.Secret("synthetic-wrong-password")
		},
		"database missing": func(t *testing.T, cfg *config.DatabaseConfig) {
			cfg.Name = dbtest.NewName(t)
		},
	}
	for name, breakConfig := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := dbtest.ServerConfig(t)
			secret := cfg.Password.Reveal()
			breakConfig(t, &cfg)
			h := router(t, cfg)

			start := time.Now()
			code, body := get(t, h, "/readyz")
			if code != http.StatusServiceUnavailable {
				t.Errorf("/readyz = %d, want 503", code)
			}
			if elapsed := time.Since(start); elapsed > httpserver.DefaultReadinessTimeout+time.Second {
				t.Errorf("/readyz took %s, want at most about %s", elapsed, httpserver.DefaultReadinessTimeout)
			}
			for _, leak := range []string{secret, "synthetic-wrong-password", cfg.User, cfg.Name, "password", "connect", "SQLSTATE"} {
				if leak != "" && strings.Contains(body, leak) {
					t.Errorf("/readyz body reveals %q: %s", leak, body)
				}
			}
			if code, _ := get(t, h, "/healthz"); code != http.StatusOK {
				t.Errorf("/healthz = %d, want 200 while the database is unavailable", code)
			}
		})
	}
}
