package config

import (
	"errors"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
)

func defaultSignInLimit() SignInLimitConfig {
	return SignInLimitConfig{
		FailuresPerIPv4: DefaultSignInFailuresIPv4, FailuresPerIPv6Subnet: DefaultSignInFailuresIPv6Subnet,
		FailuresPerIPv6Aggregate: DefaultSignInFailuresIPv6Aggregate, FailureWindow: DefaultSignInFailureWindow,
		RequestsPerSecond: DefaultSignInRequestRate, RequestBurst: DefaultSignInRequestBurst,
	}
}

func defaultPasswordHash() PasswordHashConfig {
	return PasswordHashConfig{Concurrency: DefaultPasswordHashConcurrency, Queue: DefaultPasswordHashQueue, MaxWait: DefaultPasswordHashMaxWait}
}

func TestLoadSecuritySettings(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(map[string]string{
		EnvTrustedProxies:              " 10.0.0.0/8, 192.0.2.10 ,2001:db8::/48, 2001:db8::1, 10.1.2.3/8, 2001:db8::/32",
		EnvSignInFailuresIPv4:          "20",
		EnvSignInFailuresIPv6Subnet:    "30",
		EnvSignInFailuresIPv6Aggregate: "300",
		EnvSignInFailureWindow:         "10m",
		EnvSignInRequestRate:           "2.5",
		EnvSignInRequestBurst:          "40",
		EnvPasswordHashConcurrency:     "8",
		EnvPasswordHashQueue:           "0",
		EnvPasswordHashMaxWait:         "250ms",
	}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	want := TrustedProxyConfig{Set: true, Prefixes: []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.0.2.10/32"),
		netip.MustParsePrefix("2001:db8::/48"),
		netip.MustParsePrefix("2001:db8::1/128"),
		netip.MustParsePrefix("10.0.0.0/8"), // masked
		netip.MustParsePrefix("2001:db8::/32"),
	}}
	if !reflect.DeepEqual(cfg.HTTP.Proxies, want) {
		t.Errorf("Proxies = %+v, want %+v", cfg.HTTP.Proxies, want)
	}
	wantLimit := SignInLimitConfig{FailuresPerIPv4: 20, FailuresPerIPv6Subnet: 30, FailuresPerIPv6Aggregate: 300,
		FailureWindow: 10 * time.Minute, RequestsPerSecond: 2.5, RequestBurst: 40}
	if cfg.SignInLimit != wantLimit {
		t.Errorf("SignInLimit = %+v, want %+v", cfg.SignInLimit, wantLimit)
	}
	if cfg.PasswordHash != (PasswordHashConfig{Concurrency: 8, Queue: 0, MaxWait: 250 * time.Millisecond}) {
		t.Errorf("PasswordHash = %+v", cfg.PasswordHash)
	}

	// Defaults: TRUSTED_PROXIES not set, the owner-approved initial limits.
	def, err := Load(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if def.HTTP.Proxies.Set || def.HTTP.Proxies.Prefixes != nil {
		t.Errorf("default proxies = %+v", def.HTTP.Proxies)
	}
	if def.SignInLimit != (SignInLimitConfig{FailuresPerIPv4: 100, FailuresPerIPv6Subnet: 100, FailuresPerIPv6Aggregate: 500,
		FailureWindow: 15 * time.Minute, RequestsPerSecond: 10, RequestBurst: 100}) {
		t.Errorf("default limit = %+v", def.SignInLimit)
	}
	if def.PasswordHash != (PasswordHashConfig{Concurrency: 4, Queue: 32, MaxWait: time.Second}) {
		t.Errorf("default password hash limits = %+v", def.PasswordHash)
	}
}

func TestTrustedProxiesNone(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"none", " NONE "} {
		cfg, err := Load(env(map[string]string{EnvTrustedProxies: v}))
		if err != nil {
			t.Fatalf("%q: %v", v, err)
		}
		if !cfg.HTTP.Proxies.Set || len(cfg.HTTP.Proxies.Prefixes) != 0 {
			t.Errorf("%q: Proxies = %+v, want set and empty", v, cfg.HTTP.Proxies)
		}
		cfg.Env = EnvironmentProduction
		if err := cfg.ValidateAPI(); err != nil {
			t.Errorf("%q in production: %v", v, err)
		}
	}
}

func TestValidateAPIRequiresTrustedProxiesInProduction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		env     Environment
		proxies TrustedProxyConfig
		wantErr bool
	}{
		{"production unset", EnvironmentProduction, TrustedProxyConfig{}, true},
		{"production none", EnvironmentProduction, TrustedProxyConfig{Set: true}, false},
		{"production proxies", EnvironmentProduction, TrustedProxyConfig{Set: true, Prefixes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}}, false},
		{"development unset", EnvironmentDevelopment, TrustedProxyConfig{}, false},
		{"test unset", EnvironmentTest, TrustedProxyConfig{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := Config{Env: tc.env, HTTP: HTTPConfig{Proxies: tc.proxies}}.ValidateAPI()
			if (err != nil) != tc.wantErr || (err != nil && !errors.Is(err, ErrTrustedProxiesRequired)) {
				t.Errorf("ValidateAPI() = %v, want error %v", err, tc.wantErr)
			}
		})
	}
	// Blank counts as unset.
	cfg, err := Load(env(map[string]string{EnvTrustedProxies: "   "}))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Env = EnvironmentProduction
	if !errors.Is(cfg.ValidateAPI(), ErrTrustedProxiesRequired) {
		t.Error("blank TRUSTED_PROXIES accepted in production")
	}
}

func TestBroadTrustedProxies(t *testing.T) {
	t.Parallel()
	broad := "1.0.0.0/7, 2001::/16"
	if _, err := Load(env(map[string]string{EnvTrustedProxies: broad})); err == nil || !strings.Contains(err.Error(), EnvTrustedProxiesAllowBroad) {
		t.Fatalf("broad ranges without opt-in: error = %v, want one naming %s", err, EnvTrustedProxiesAllowBroad)
	}
	cfg, err := Load(env(map[string]string{EnvTrustedProxies: broad + ", 192.0.2.1", EnvTrustedProxiesAllowBroad: "true"}))
	if err != nil {
		t.Fatalf("broad ranges with opt-in: %v", err)
	}
	wantBroad := []netip.Prefix{netip.MustParsePrefix("0.0.0.0/7"), netip.MustParsePrefix("2001::/16")}
	if !reflect.DeepEqual(cfg.HTTP.Proxies.Broad, wantBroad) || len(cfg.HTTP.Proxies.Prefixes) != 3 {
		t.Errorf("Proxies = %+v, want broad %v", cfg.HTTP.Proxies, wantBroad)
	}
}

func TestLoadRejectsInvalidSecuritySettings(t *testing.T) {
	t.Parallel()

	allowBroad := func(v string) map[string]string {
		return map[string]string{EnvTrustedProxies: v, EnvTrustedProxiesAllowBroad: "true"}
	}
	tests := []struct {
		name string
		vars map[string]string
		key  string
	}{
		{"host name proxy", map[string]string{EnvTrustedProxies: "proxy.example.invalid"}, EnvTrustedProxies},
		{"bad prefix", map[string]string{EnvTrustedProxies: "10.0.0.0/33"}, EnvTrustedProxies},
		{"malformed address", map[string]string{EnvTrustedProxies: "10.0.0.256"}, EnvTrustedProxies},
		{"malformed prefix length", map[string]string{EnvTrustedProxies: "10.0.0.0/x"}, EnvTrustedProxies},
		{"empty entry", map[string]string{EnvTrustedProxies: "10.0.0.1,,10.0.0.2"}, EnvTrustedProxies},
		{"none with addresses", map[string]string{EnvTrustedProxies: "none,10.0.0.1"}, EnvTrustedProxies},
		{"trust everything (IPv4)", map[string]string{EnvTrustedProxies: "0.0.0.0/0"}, EnvTrustedProxies},
		{"trust everything (IPv6)", map[string]string{EnvTrustedProxies: "::/0"}, EnvTrustedProxies},
		{"trust everything even with opt-in (IPv4)", allowBroad("0.0.0.0/0"), EnvTrustedProxies},
		{"broader than /8", map[string]string{EnvTrustedProxies: "10.0.0.0/7"}, EnvTrustedProxiesAllowBroad},
		{"broader than /32", map[string]string{EnvTrustedProxies: "2001:db8::/31"}, EnvTrustedProxiesAllowBroad},
		{"IPv4 halves", map[string]string{EnvTrustedProxies: "0.0.0.0/1,128.0.0.0/1"}, EnvTrustedProxies},
		{"IPv6 halves", map[string]string{EnvTrustedProxies: "::/1,8000::/1"}, EnvTrustedProxies},
		{"IPv4 halves with opt-in", allowBroad("0.0.0.0/1,128.0.0.0/1"), EnvTrustedProxies},
		{"IPv6 halves with opt-in", allowBroad("::/1,8000::/1"), EnvTrustedProxies},
		{"IPv4 quarters with opt-in", allowBroad("0.0.0.0/2,64.0.0.0/2,128.0.0.0/2,192.0.0.0/2"), EnvTrustedProxies},
		{"IPv4 uneven cover with opt-in", allowBroad("0.0.0.0/1,128.0.0.0/2,192.0.0.0/3,224.0.0.0/3"), EnvTrustedProxies},
		{"zone", map[string]string{EnvTrustedProxies: "fe80::1%eth0"}, EnvTrustedProxies},
		{"mapped IPv4", map[string]string{EnvTrustedProxies: "::ffff:10.0.0.1"}, EnvTrustedProxies},
		{"too many proxies", map[string]string{EnvTrustedProxies: strings.TrimSuffix(strings.Repeat("10.0.0.1,", 65), ",")}, EnvTrustedProxies},
		{"opt-in not a boolean", map[string]string{EnvTrustedProxiesAllowBroad: "yes"}, EnvTrustedProxiesAllowBroad},
		{"zero failures", map[string]string{EnvSignInFailuresIPv4: "0"}, EnvSignInFailuresIPv4},
		{"too many failures", map[string]string{EnvSignInFailuresIPv6Subnet: "1000001"}, EnvSignInFailuresIPv6Subnet},
		{"aggregate below subnet", map[string]string{EnvSignInFailuresIPv6Aggregate: "50"}, EnvSignInFailuresIPv6Aggregate},
		{"failures not a number", map[string]string{EnvSignInFailuresIPv4: "many"}, EnvSignInFailuresIPv4},
		{"window too short", map[string]string{EnvSignInFailureWindow: "30s"}, EnvSignInFailureWindow},
		{"window too long", map[string]string{EnvSignInFailureWindow: "25h"}, EnvSignInFailureWindow},
		{"window not a duration", map[string]string{EnvSignInFailureWindow: "15"}, EnvSignInFailureWindow},
		{"zero rate", map[string]string{EnvSignInRequestRate: "0"}, EnvSignInRequestRate},
		{"negative rate", map[string]string{EnvSignInRequestRate: "-1"}, EnvSignInRequestRate},
		{"NaN rate", map[string]string{EnvSignInRequestRate: "NaN"}, EnvSignInRequestRate},
		{"zero burst", map[string]string{EnvSignInRequestBurst: "0"}, EnvSignInRequestBurst},
		{"zero hash concurrency", map[string]string{EnvPasswordHashConcurrency: "0"}, EnvPasswordHashConcurrency},
		{"negative hash queue", map[string]string{EnvPasswordHashQueue: "-1"}, EnvPasswordHashQueue},
		{"hash wait too long", map[string]string{EnvPasswordHashMaxWait: "2m"}, EnvPasswordHashMaxWait},
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

func TestCoversPrefix(t *testing.T) {
	t.Parallel()
	all4 := netip.MustParsePrefix("0.0.0.0/0")
	parse := func(ss ...string) []netip.Prefix {
		out := make([]netip.Prefix, len(ss))
		for i, s := range ss {
			out[i] = netip.MustParsePrefix(s)
		}
		return out
	}
	if coversPrefix(parse("0.0.0.0/1", "128.0.0.0/2", "192.0.0.0/3"), all4) {
		t.Error("a gap (224.0.0.0/3) reported as covered")
	}
	if !coversPrefix(parse("0.0.0.0/1", "128.0.0.0/2", "192.0.0.0/3", "224.0.0.0/3"), all4) {
		t.Error("a full cover reported as not covered")
	}
	if coversPrefix(parse("10.0.0.0/8", "2001:db8::/32"), all4) {
		t.Error("a small list reported as covering IPv4")
	}
	if !coversPrefix(parse("10.0.0.0/8"), netip.MustParsePrefix("10.1.0.0/16")) {
		t.Error("a containing prefix not detected")
	}
}
