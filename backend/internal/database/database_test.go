package database

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/config"
)

// syntheticPassword is not a real credential; tests assert it never appears in errors.
const syntheticPassword = `synthetic pa55 'quoted' \back\slash`

func testConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:            "127.0.0.1",
		Port:            5432,
		Name:            "sms",
		User:            "sms",
		Password:        config.Secret(syntheticPassword),
		SSLMode:         config.SSLModeDisable,
		MaxConns:        7,
		MinConns:        0,
		ConnectTimeout:  3 * time.Second,
		MaxConnLifetime: 45 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
	}
}

func TestPoolConfigTranslatesSettings(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	pc, err := poolConfig(cfg, "sms-test")
	if err != nil {
		t.Fatalf("poolConfig() error: %v", err)
	}
	cc := pc.ConnConfig
	if cc.Host != cfg.Host || int(cc.Port) != cfg.Port || cc.Database != cfg.Name || cc.User != cfg.User {
		t.Errorf("connection target = %s:%d/%s as %s", cc.Host, cc.Port, cc.Database, cc.User)
	}
	if cc.Password != syntheticPassword {
		t.Error("password with spaces, quotes, and backslashes must be passed to pgx unchanged")
	}
	if cc.RuntimeParams["application_name"] != "sms-test" {
		t.Errorf("application_name = %q, want sms-test", cc.RuntimeParams["application_name"])
	}
	if cc.ConnectTimeout != cfg.ConnectTimeout {
		t.Errorf("ConnectTimeout = %s, want %s", cc.ConnectTimeout, cfg.ConnectTimeout)
	}
	if pc.MaxConns != 7 || pc.MinConns != 0 || pc.MaxConnLifetime != cfg.MaxConnLifetime || pc.MaxConnIdleTime != cfg.MaxConnIdleTime {
		t.Errorf("pool limits = max %d min %d lifetime %s idle %s", pc.MaxConns, pc.MinConns, pc.MaxConnLifetime, pc.MaxConnIdleTime)
	}
	if cc.TLSConfig != nil {
		t.Error("sslmode=disable must not configure TLS")
	}
}

func TestPoolConfigTLSModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode           config.SSLMode
		wantTLS        bool
		wantVerifyHost bool
	}{
		{config.SSLModeDisable, false, false},
		{config.SSLModeRequire, true, false},
		{config.SSLModeVerifyFull, true, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			t.Parallel()
			cfg := testConfig()
			cfg.Host = "db.example.invalid"
			cfg.SSLMode = tt.mode
			pc, err := poolConfig(cfg, "sms-test")
			if err != nil {
				t.Fatalf("poolConfig() error: %v", err)
			}
			tlsCfg := pc.ConnConfig.TLSConfig
			if (tlsCfg != nil) != tt.wantTLS {
				t.Fatalf("TLS configured = %v, want %v", tlsCfg != nil, tt.wantTLS)
			}
			if len(pc.ConnConfig.Fallbacks) != 0 {
				t.Errorf("sslmode=%s must not fall back to another connection mode", tt.mode)
			}
			if tt.wantVerifyHost && (tlsCfg.ServerName != cfg.Host || tlsCfg.InsecureSkipVerify) {
				t.Errorf("verify-full must verify the certificate for %s; got ServerName=%q InsecureSkipVerify=%v",
					cfg.Host, tlsCfg.ServerName, tlsCfg.InsecureSkipVerify)
			}
		})
	}
}

// closedPort returns a loopback port with nothing listening on it.
func closedPort(t *testing.T) int {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return port
}

func TestPingUnreachableDatabase(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Port = closedPort(t)
	db, err := Open(context.Background(), cfg, "sms-test")
	if err != nil {
		t.Fatalf("Open() must not connect, but returned: %v", err)
	}
	t.Cleanup(db.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.Ping(ctx)
	if err == nil {
		t.Fatal("Ping() against a closed port succeeded")
	}
	if strings.Contains(err.Error(), syntheticPassword) || strings.Contains(err.Error(), "pa55") {
		t.Errorf("error reveals the password: %v", err)
	}
}

func TestPingHonoursContextCancellation(t *testing.T) {
	t.Parallel()

	// A listener that accepts connections but never answers, like a hung server.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	var (
		mu    sync.Mutex
		conns []net.Conn
		done  = make(chan struct{})
	)
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed during cleanup
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		<-done
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		}
	})

	cfg := testConfig()
	cfg.Port = ln.Addr().(*net.TCPAddr).Port
	cfg.ConnectTimeout = time.Minute // the context deadline must win
	db, err := Open(context.Background(), cfg, "sms-test")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(db.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = db.Ping(ctx)
	if err == nil {
		t.Fatal("Ping() against a hung server succeeded")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Ping() took %s; it must return when the context expires", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Ping() error = %v, want a deadline error", err)
	}
}

func TestPingAfterClose(t *testing.T) {
	t.Parallel()

	db, err := Open(context.Background(), testConfig(), "sms-test")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	db.Close()
	if err := db.Ping(context.Background()); err == nil {
		t.Error("Ping() on a closed pool succeeded")
	}
}

func TestQuote(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"":            `''`,
		"plain":       `'plain'`,
		"with space":  `'with space'`,
		`it's`:        `'it\'s'`,
		`back\slash`:  `'back\\slash'`,
		`\'`:          `'\\\''`,
		"key=val x=y": `'key=val x=y'`,
	} {
		if got := quote(in); got != want {
			t.Errorf("quote(%q) = %s, want %s", in, got, want)
		}
	}
}
