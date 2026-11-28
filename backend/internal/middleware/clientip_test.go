package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIP(t *testing.T) {
	t.Parallel()
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("2001:db8:ff::/48")}
	tests := []struct {
		name    string
		trusted []netip.Prefix
		remote  string
		xff     []string
		want    string // "" for invalid
	}{
		{"direct peer", nil, "203.0.113.7:51234", nil, "203.0.113.7"},
		{"IPv6 peer", nil, "[2001:db8::5]:443", nil, "2001:db8::5"},
		{"IPv4-mapped peer", nil, "[::ffff:203.0.113.7]:1", nil, "203.0.113.7"},
		{"IPv6 zone removed", nil, "[fe80::1%eth0]:1", nil, "fe80::1"},
		{"unparseable peer", nil, "not-an-address", nil, ""},

		// Without trusted proxies, X-Forwarded-For is never read (spoofing attempt).
		{"spoofed header, no trusted proxy", nil, "203.0.113.7:1", []string{"198.51.100.1"}, "203.0.113.7"},
		// A peer that is not a trusted proxy cannot name another client either.
		{"spoofed header from untrusted peer", trusted, "203.0.113.7:1", []string{"198.51.100.1"}, "203.0.113.7"},

		// Behind trusted proxies: the rightmost address that is not a trusted proxy.
		{"one trusted proxy", trusted, "10.0.0.2:1", []string{"198.51.100.1"}, "198.51.100.1"},
		{"client-supplied entry left of the real client", trusted, "10.0.0.2:1", []string{"6.6.6.6, 198.51.100.1"}, "198.51.100.1"},
		{"chain of trusted proxies", trusted, "10.0.0.2:1", []string{"198.51.100.1, 10.1.1.1", "10.2.2.2"}, "198.51.100.1"},
		{"IPv6 trusted proxy", trusted, "[2001:db8:ff::3]:1", []string{"2001:db8:1::9"}, "2001:db8:1::9"},
		{"only trusted proxies", trusted, "10.0.0.2:1", []string{"10.9.9.9"}, "10.9.9.9"},
		{"no header from trusted proxy", trusted, "10.0.0.2:1", nil, "10.0.0.2"},
		{"malformed entry stops the search", trusted, "10.0.0.2:1", []string{"198.51.100.1, garbage"}, "10.0.0.2"},
		{"malformed entry after a trusted hop", trusted, "10.0.0.2:1", []string{"evil, 10.3.3.3"}, "10.3.3.3"},
		{"entry with port is malformed", trusted, "10.0.0.2:1", []string{"198.51.100.1:80"}, "10.0.0.2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got netip.Addr
			h := ClientIP(tc.trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = ClientIPFromContext(r.Context())
			}))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				req.Header.Add("X-Forwarded-For", v)
			}
			req.Header.Set("X-Real-IP", "192.0.2.99") // never used
			req.Header.Set("Forwarded", "for=192.0.2.98")
			h.ServeHTTP(httptest.NewRecorder(), req)
			if tc.want == "" {
				if got.IsValid() {
					t.Errorf("client IP = %s, want invalid", got)
				}
				return
			}
			if got != netip.MustParseAddr(tc.want) {
				t.Errorf("client IP = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestClientIPBoundsForwardedHops(t *testing.T) {
	t.Parallel()
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	var got netip.Addr
	h := ClientIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = ClientIPFromContext(r.Context())
	}))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1"
	req.Header.Add("X-Forwarded-For", "198.51.100.1")
	for range 100 {
		req.Header.Add("X-Forwarded-For", "10.0.0.9")
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got != netip.MustParseAddr("10.0.0.9") {
		t.Errorf("client IP = %s, want the last trusted hop examined (10.0.0.9)", got)
	}
	if ClientIPFromContext(context.Background()).IsValid() {
		t.Error("ClientIPFromContext without the middleware returned an address")
	}
}
