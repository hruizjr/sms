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

func TestBudgets(t *testing.T) {
	t.Parallel()
	l := &Limiter{cfg: DefaultConfig}
	tests := []struct {
		ip   string
		want []budget
	}{
		{"203.0.113.7", []budget{{netip.MustParsePrefix("203.0.113.7/32"), 100}}},
		{"::ffff:203.0.113.7", []budget{{netip.MustParsePrefix("203.0.113.7/32"), 100}}},
		// The /48 aggregate first (lock order), then the /64.
		{"2001:db8:1:2:3:4:5:6", []budget{{netip.MustParsePrefix("2001:db8:1::/48"), 500}, {netip.MustParsePrefix("2001:db8:1:2::/64"), 100}}},
		{"2001:db8:1:ffff::1", []budget{{netip.MustParsePrefix("2001:db8:1::/48"), 500}, {netip.MustParsePrefix("2001:db8:1:ffff::/64"), 100}}},
	}
	for _, tc := range tests {
		got := l.budgets(netip.MustParseAddr(tc.ip))
		if len(got) != len(tc.want) {
			t.Fatalf("budgets(%s) = %v, want %v", tc.ip, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("budgets(%s)[%d] = %v, want %v", tc.ip, i, got[i], tc.want[i])
			}
		}
	}
	if got := l.budgets(netip.Addr{}); len(got) != 1 || got[0].source != netip.MustParsePrefix("::/128") || got[0].limit != 100 {
		t.Errorf("budgets(invalid) = %v", got)
	}
}

func TestWholeSeconds(t *testing.T) {
	t.Parallel()
	for in, want := range map[time.Duration]time.Duration{
		0: time.Second, -time.Second: time.Second, time.Millisecond: time.Second,
		time.Second: time.Second, 1500 * time.Millisecond: 2 * time.Second, 90 * time.Second: 90 * time.Second,
	} {
		if got := wholeSeconds(in); got != want {
			t.Errorf("wholeSeconds(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestNewRejectsInvalidLimits(t *testing.T) {
	t.Parallel()
	mutate := func(f func(*Config)) Config {
		c := DefaultConfig
		f(&c)
		return c
	}
	for _, cfg := range []Config{
		mutate(func(c *Config) { c.FailuresPerIPv4 = 0 }),
		mutate(func(c *Config) { c.FailuresPerIPv6Subnet = -1 }),
		mutate(func(c *Config) { c.FailuresPerIPv6Aggregate = 0 }),
		mutate(func(c *Config) { c.FailureWindow = 0 }),
		mutate(func(c *Config) { c.RequestsPerSecond = 0 }),
		mutate(func(c *Config) { c.RequestBurst = 0 }),
		mutate(func(c *Config) { c.RequestBurst = maxBurst + 1 }),
	} {
		if _, err := New(nil, cfg); err == nil {
			t.Errorf("New(%+v) succeeded", cfg)
		}
	}
	if err := DefaultConfig.Validate(); err != nil {
		t.Errorf("DefaultConfig: %v", err)
	}
}
