package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// maxHeaderBytes bounds request header size on both listeners.
const maxHeaderBytes = 64 << 10 // 64 KiB

// ServerConfig holds listener addresses and timeouts. It is built from config.Config in main;
// tests may use port 0 through Serve with their own listeners.
type ServerConfig struct {
	Addr              string
	MetricsAddr       string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// Server runs the public application listener and the separate metrics listener.
type Server struct {
	cfg       ServerConfig
	logger    *slog.Logger
	app       http.Handler
	metrics   http.Handler
	readiness *Readiness
}

// NewServer creates a Server. app serves the API; metrics serves Prometheus metrics on the
// metrics listener only.
func NewServer(cfg ServerConfig, logger *slog.Logger, app, metrics http.Handler, readiness *Readiness) *Server {
	return &Server{cfg: cfg, logger: logger, app: app, metrics: metrics, readiness: readiness}
}

// Run binds both listeners and serves until ctx is cancelled, then shuts down gracefully.
// A bind failure on either listener is returned immediately.
func (s *Server) Run(ctx context.Context) error {
	var lc net.ListenConfig
	appLn, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen on application address %s: %w", s.cfg.Addr, err)
	}
	metricsLn, err := lc.Listen(ctx, "tcp", s.cfg.MetricsAddr)
	if err != nil {
		closeErr := appLn.Close()
		return errors.Join(fmt.Errorf("listen on metrics address %s: %w", s.cfg.MetricsAddr, err), closeErr)
	}
	return s.Serve(ctx, appLn, metricsLn)
}

// Serve serves on the given listeners until ctx is cancelled or a listener fails, then shuts
// both servers down. It owns the listeners and both serving goroutines, and returns only after
// both goroutines have exited.
//
// Shutdown order: mark not ready → stop the application server (waiting up to ShutdownTimeout
// for in-flight requests) → stop the metrics server. If the deadline passes, remaining
// connections are closed forcibly and the deadline error is returned.
func (s *Server) Serve(ctx context.Context, appLn, metricsLn net.Listener) error {
	appSrv := s.newHTTPServer(s.app)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", s.metrics)
	metricsSrv := s.newHTTPServer(metricsMux)

	type result struct {
		name string
		err  error
	}
	results := make(chan result, 2)
	serve := func(name string, srv *http.Server, ln net.Listener) {
		err := srv.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		results <- result{name: name, err: err}
	}
	go serve("application", appSrv, appLn)
	go serve("metrics", metricsSrv, metricsLn)
	pending := 2

	s.readiness.SetReady(true)
	s.logger.Info("api started",
		slog.String("http_addr", appLn.Addr().String()),
		slog.String("metrics_addr", metricsLn.Addr().String()))

	var errs []error
	select {
	case <-ctx.Done():
		s.logger.Info("shutdown requested")
	case res := <-results:
		pending--
		s.logger.Error("listener stopped unexpectedly", slog.String("listener", res.name), slog.Any("error", res.err))
		errs = append(errs, fmt.Errorf("%s server stopped: %w", res.name, errorOrUnexpected(res.err)))
	}

	s.readiness.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.ShutdownTimeout)
	defer cancel()
	for _, srv := range []struct {
		name string
		srv  *http.Server
	}{{"application", appSrv}, {"metrics", metricsSrv}} {
		if err := srv.srv.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("graceful shutdown of %s server: %w", srv.name, err))
			if closeErr := srv.srv.Close(); closeErr != nil {
				errs = append(errs, fmt.Errorf("force close of %s server: %w", srv.name, closeErr))
			}
		}
	}

	for ; pending > 0; pending-- {
		if res := <-results; res.err != nil {
			errs = append(errs, fmt.Errorf("%s server: %w", res.name, res.err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}
	s.logger.Info("api stopped")
	return nil
}

func (s *Server) newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadTimeout:       s.cfg.ReadTimeout,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(s.logger.Handler(), slog.LevelWarn),
	}
}

// errorOrUnexpected turns a nil error from a server that stopped on its own into a real error.
func errorOrUnexpected(err error) error {
	if err == nil {
		return errors.New("stopped without an error")
	}
	return err
}
