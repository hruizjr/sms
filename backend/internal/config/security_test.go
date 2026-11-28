package config

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadSecuritySettings(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{
		EnvTrustedProxies:    " 10.0.0.0/8, 192.0.2.10 ,2001:db8::/48, 2001:db8::1, 10.1.2.3/8",
		EnvSignInMaxAttempts: "20",
		EnvSignInWindow:      "10m",
	}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	want := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.0.2.10/32"),
		netip.MustParsePrefix("2001:db8::/48"),
		netip.MustParsePrefix("2001:db8::1/128"),
		netip.MustParsePrefix("10.0.0.0/8"), // masked
	}
	if !reflect.DeepEqual(cfg.HTTP.TrustedProxies, want) {
		t.Errorf("TrustedProxies = %v, want %v", cfg.HTTP.TrustedProxies, want)
	}
	if cfg.SignInLimit != (SignInLimitConfig{MaxAttempts: 20, Window: 10 * time.Minute}) {
		t.Errorf("SignInLimit = %+v", cfg.SignInLimit)
	}

	// Defaults: no trusted proxy, the documented limit.
	def, err := Load(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if def.HTTP.TrustedProxies != nil || def.SignInLimit != (SignInLimitConfig{MaxAttempts: 50, Window: 15 * time.Minute}) {
		t.Errorf("defaults: proxies %v, limit %+v", def.HTTP.TrustedProxies, def.SignInLimit)
	}
}

func TestLoadRejectsInvalidSecuritySettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vars map[string]string
		key  string
	}{
		{"host name proxy", map[string]string{EnvTrustedProxies: "proxy.example.invalid"}, EnvTrustedProxies},
		{"bad prefix", map[string]string{EnvTrustedProxies: "10.0.0.0/33"}, EnvTrustedProxies},
		{"trust everything (IPv4)", map[string]string{EnvTrustedProxies: "0.0.0.0/0"}, EnvTrustedProxies},
		{"trust everything (IPv6)", map[string]string{EnvTrustedProxies: "::/0"}, EnvTrustedProxies},
		{"zone", map[string]string{EnvTrustedProxies: "fe80::1%eth0"}, EnvTrustedProxies},
		{"mapped IPv4", map[string]string{EnvTrustedProxies: "::ffff:10.0.0.1"}, EnvTrustedProxies},
		{"too many proxies", map[string]string{EnvTrustedProxies: strings.Repeat("10.0.0.1,", 65)}, EnvTrustedProxies},
		{"zero attempts", map[string]string{EnvSignInMaxAttempts: "0"}, EnvSignInMaxAttempts},
		{"too many attempts", map[string]string{EnvSignInMaxAttempts: "100001"}, EnvSignInMaxAttempts},
		{"not a number", map[string]string{EnvSignInMaxAttempts: "many"}, EnvSignInMaxAttempts},
		{"window too short", map[string]string{EnvSignInWindow: "30s"}, EnvSignInWindow},
		{"window too long", map[string]string{EnvSignInWindow: "25h"}, EnvSignInWindow},
		{"window not a duration", map[string]string{EnvSignInWindow: "15"}, EnvSignInWindow},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(env(tc.vars))
			if err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Errorf("Load() error = %v, want one naming %s", err, tc.key)
			}
		})
	}
}
