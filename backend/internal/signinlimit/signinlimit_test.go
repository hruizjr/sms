package signinlimit

import (
	"net/netip"
	"testing"
	"time"
)

func TestSource(t *testing.T) {
	t.Parallel()
	tests := []struct{ ip, want string }{
		{"203.0.113.7", "203.0.113.7/32"},
		{"::ffff:203.0.113.7", "203.0.113.7/32"}, // IPv4-mapped IPv6 is the IPv4 address
		{"2001:db8:1:2:3:4:5:6", "2001:db8:1:2::/64"},
		{"2001:db8:1:2:ffff::1", "2001:db8:1:2::/64"}, // same /64, same source
		{"fe80::1%eth0", "fe80::/64"},
	}
	for _, tc := range tests {
		if got := Source(netip.MustParseAddr(tc.ip)); got != netip.MustParsePrefix(tc.want) {
			t.Errorf("Source(%s) = %s, want %s", tc.ip, got, tc.want)
		}
	}
	if got := Source(netip.Addr{}); got != netip.MustParsePrefix("::/128") {
		t.Errorf("Source(invalid) = %s, want ::/128", got)
	}
}

func TestNewRejectsInvalidLimits(t *testing.T) {
	t.Parallel()
	for _, cfg := range []Config{{0, time.Minute}, {-1, time.Minute}, {5, 0}, {5, -time.Second}} {
		if _, err := New(nil, cfg); err == nil {
			t.Errorf("New(%+v) succeeded", cfg)
		}
	}
}
