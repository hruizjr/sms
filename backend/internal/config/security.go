package config

import (
	"errors"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Security environment variables (Phase 4 Group 6). Documented in backend/README.md and
// backend/.env.example.
const (
	// EnvTrustedProxies lists the reverse proxies or load balancers whose X-Forwarded-For header is
	// believed: comma-separated IP addresses or CIDR prefixes, or "none" when the API is reached
	// directly. In production it MUST be set (the API refuses to start without it, ValidateAPI);
	// elsewhere unset means none.
	EnvTrustedProxies = "TRUSTED_PROXIES"
	// EnvTrustedProxiesAllowBroad ("true") accepts TRUSTED_PROXIES entries broader than an IPv4 /8 or
	// an IPv6 /32. Off by default; the API logs a warning at startup when it is used.
	EnvTrustedProxiesAllowBroad = "TRUSTED_PROXIES_ALLOW_BROAD_RANGES"

	// Failed-attempt budgets of the portal sign-in (internal/signinlimit).
	EnvSignInFailuresIPv4          = "SIGNIN_FAILURE_LIMIT_IPV4"
	EnvSignInFailuresIPv6Subnet    = "SIGNIN_FAILURE_LIMIT_IPV6_64"
	EnvSignInFailuresIPv6Aggregate = "SIGNIN_FAILURE_LIMIT_IPV6_48"
	EnvSignInFailureWindow         = "SIGNIN_FAILURE_WINDOW"
	// All-attempt token bucket of the portal sign-in.
	EnvSignInRequestRate  = "SIGNIN_REQUEST_RATE"
	EnvSignInRequestBurst = "SIGNIN_REQUEST_BURST"

	// Password-hashing capacity (internal/portal.HashLimits).
	EnvPasswordHashConcurrency = "PASSWORD_HASH_CONCURRENCY"
	EnvPasswordHashQueue       = "PASSWORD_HASH_QUEUE"
	EnvPasswordHashMaxWait     = "PASSWORD_HASH_MAX_WAIT"
)

// TrustedProxiesNone is the TRUSTED_PROXIES value that declares that there is no reverse proxy.
const TrustedProxiesNone = "none"

// Initial defaults (owner decision, decision log §23); explained in backend/README.md. They are
// configurable and are reviewed against measured traffic before launch.
const (
	DefaultSignInFailuresIPv4          = 100
	DefaultSignInFailuresIPv6Subnet    = 100
	DefaultSignInFailuresIPv6Aggregate = 500
	DefaultSignInFailureWindow         = 15 * time.Minute
	DefaultSignInRequestRate           = 10.0
	DefaultSignInRequestBurst          = 100

	DefaultPasswordHashConcurrency = 4
	DefaultPasswordHashQueue       = 32
	DefaultPasswordHashMaxWait     = time.Second

	MaxSignInFailures      = 1_000_000
	MinSignInFailureWindow = time.Minute
	MaxSignInFailureWindow = 24 * time.Hour
	MaxSignInRequestRate   = 10_000.0
	MaxSignInRequestBurst  = 1_000_000

	MaxPasswordHashConcurrency = 256
	MaxPasswordHashQueue       = 100_000
	MaxPasswordHashMaxWait     = time.Minute

	// maxTrustedProxies bounds the list (each request is checked against every entry).
	maxTrustedProxies = 64
	// The broadest prefixes accepted without EnvTrustedProxiesAllowBroad.
	minTrustedIPv4Bits = 8
	minTrustedIPv6Bits = 32
)

// SignInLimitConfig is the layered per-source sign-in limit.
type SignInLimitConfig struct {
	FailuresPerIPv4          int
	FailuresPerIPv6Subnet    int
	FailuresPerIPv6Aggregate int
	FailureWindow            time.Duration
	RequestsPerSecond        float64
	RequestBurst             int
}

// PasswordHashConfig bounds password hashing (portal.HashLimits).
type PasswordHashConfig struct {
	Concurrency int
	Queue       int
	MaxWait     time.Duration
}

// TrustedProxyConfig is the parsed TRUSTED_PROXIES setting.
type TrustedProxyConfig struct {
	// Set is false when TRUSTED_PROXIES is unset or blank (refused in production by ValidateAPI).
	Set bool
	// Prefixes are the trusted proxies; empty for "none".
	Prefixes []netip.Prefix
	// Broad are the entries broader than an IPv4 /8 or an IPv6 /32 (accepted only with
	// TRUSTED_PROXIES_ALLOW_BROAD_RANGES=true; the API warns about them at startup).
	Broad []netip.Prefix
}

// ErrTrustedProxiesRequired: TRUSTED_PROXIES is not set in production.
var ErrTrustedProxiesRequired = errors.New(EnvTrustedProxies + ": must be set in production: " +
	"the addresses or CIDR prefixes of the reverse proxies or load balancers in front of the API, " +
	"or \"" + TrustedProxiesNone + "\" when clients connect to the API directly")

// ValidateAPI applies the checks that only the API process needs: in production, TRUSTED_PROXIES
// must be set explicitly, so that a deployment behind a load balancer cannot silently treat every
// client as the load balancer's address (one shared sign-in limit for all users).
func (c Config) ValidateAPI() error {
	if c.Env == EnvironmentProduction && !c.HTTP.Proxies.Set {
		return ErrTrustedProxiesRequired
	}
	return nil
}

func (r *reader) trustedProxies() TrustedProxyConfig {
	allowBroad := r.boolean(EnvTrustedProxiesAllowBroad)
	raw := r.str(EnvTrustedProxies, "")
	if raw == "" {
		return TrustedProxyConfig{}
	}
	cfg := TrustedProxyConfig{Set: true}
	if strings.EqualFold(raw, TrustedProxiesNone) {
		return cfg
	}
	for item := range strings.SplitSeq(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			r.fail(EnvTrustedProxies, "must not contain an empty entry")
			continue
		}
		if strings.EqualFold(item, TrustedProxiesNone) {
			r.fail(EnvTrustedProxies, "%q cannot be combined with addresses", TrustedProxiesNone)
			continue
		}
		var p netip.Prefix
		if strings.Contains(item, "/") {
			parsed, err := netip.ParsePrefix(item)
			if err != nil {
				r.fail(EnvTrustedProxies, "must be comma-separated IP addresses or CIDR prefixes; %q is neither", item)
				continue
			}
			p = parsed.Masked()
		} else {
			ip, err := netip.ParseAddr(item)
			if err != nil || ip.Zone() != "" {
				r.fail(EnvTrustedProxies, "must be comma-separated IP addresses or CIDR prefixes; %q is neither", item)
				continue
			}
			p = netip.PrefixFrom(ip, ip.BitLen())
		}
		if p.Addr().Is4In6() {
			r.fail(EnvTrustedProxies, "%q: use the IPv4 form", item)
			continue
		}
		// Every address would be trusted, so any client could choose its address.
		if p.Bits() == 0 {
			r.fail(EnvTrustedProxies, "must not contain %q, which trusts every address", item)
			continue
		}
		if isBroad(p) {
			if !allowBroad {
				r.fail(EnvTrustedProxies, "%q is broader than an IPv4 /%d or an IPv6 /%d; list the proxies' actual "+
					"addresses, or set %s=true to accept it", item, minTrustedIPv4Bits, minTrustedIPv6Bits, EnvTrustedProxiesAllowBroad)
				continue
			}
			cfg.Broad = append(cfg.Broad, p)
		}
		cfg.Prefixes = append(cfg.Prefixes, p)
	}
	if len(cfg.Prefixes) > maxTrustedProxies {
		r.fail(EnvTrustedProxies, "must list at most %d entries; got %d", maxTrustedProxies, len(cfg.Prefixes))
	}
	// Even with broad ranges allowed, entries that together cover a whole address family would
	// trust every client (for example 0.0.0.0/1 and 128.0.0.0/1).
	for _, root := range []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0")} {
		if coversPrefix(cfg.Prefixes, root) {
			r.fail(EnvTrustedProxies, "together the entries trust every address of %s", root)
		}
	}
	return cfg
}

func isBroad(p netip.Prefix) bool {
	if p.Addr().Is4() {
		return p.Bits() < minTrustedIPv4Bits
	}
	return p.Bits() < minTrustedIPv6Bits
}

// coversPrefix reports whether the union of prefixes contains every address of target. It splits
// target in halves only where an entry lies strictly inside it, so the work is bounded by the number
// of entries times the address length.
func coversPrefix(prefixes []netip.Prefix, target netip.Prefix) bool {
	inside := false
	for _, p := range prefixes {
		if p.Addr().BitLen() != target.Addr().BitLen() {
			continue
		}
		if p.Bits() <= target.Bits() && p.Contains(target.Addr()) {
			return true // one entry contains all of target
		}
		if p.Bits() > target.Bits() && target.Contains(p.Addr()) {
			inside = true
		}
	}
	if !inside || target.Bits() == target.Addr().BitLen() {
		return false
	}
	low, high := halves(target)
	return coversPrefix(prefixes, low) && coversPrefix(prefixes, high)
}

// halves splits p into its two prefixes one bit longer.
func halves(p netip.Prefix) (netip.Prefix, netip.Prefix) {
	bits := p.Bits() + 1
	low := netip.PrefixFrom(p.Addr(), bits)
	b := p.Addr().AsSlice()
	b[p.Bits()/8] |= 0x80 >> (p.Bits() % 8)
	highAddr, _ := netip.AddrFromSlice(b)
	return low, netip.PrefixFrom(highAddr, bits)
}

func (r *reader) signInLimit() SignInLimitConfig {
	cfg := SignInLimitConfig{
		FailuresPerIPv4:          r.integer(EnvSignInFailuresIPv4, DefaultSignInFailuresIPv4, 1, MaxSignInFailures),
		FailuresPerIPv6Subnet:    r.integer(EnvSignInFailuresIPv6Subnet, DefaultSignInFailuresIPv6Subnet, 1, MaxSignInFailures),
		FailuresPerIPv6Aggregate: r.integer(EnvSignInFailuresIPv6Aggregate, DefaultSignInFailuresIPv6Aggregate, 1, MaxSignInFailures),
		FailureWindow:            r.duration(EnvSignInFailureWindow, DefaultSignInFailureWindow),
		RequestsPerSecond:        r.positiveFloat(EnvSignInRequestRate, DefaultSignInRequestRate, MaxSignInRequestRate),
		RequestBurst:             r.integer(EnvSignInRequestBurst, DefaultSignInRequestBurst, 1, MaxSignInRequestBurst),
	}
	if cfg.FailureWindow != 0 && (cfg.FailureWindow < MinSignInFailureWindow || cfg.FailureWindow > MaxSignInFailureWindow) {
		r.fail(EnvSignInFailureWindow, "must be from %s to %s; got %s", MinSignInFailureWindow, MaxSignInFailureWindow, cfg.FailureWindow)
	}
	// The /48 aggregate contains every /64 in it, so a smaller aggregate limit would make the /64 limit
	// unreachable; that is allowed but almost certainly a mistake.
	if cfg.FailuresPerIPv6Aggregate < cfg.FailuresPerIPv6Subnet {
		r.fail(EnvSignInFailuresIPv6Aggregate, "must be at least %s (%d); got %d", EnvSignInFailuresIPv6Subnet,
			cfg.FailuresPerIPv6Subnet, cfg.FailuresPerIPv6Aggregate)
	}
	return cfg
}

func (r *reader) passwordHash() PasswordHashConfig {
	cfg := PasswordHashConfig{
		Concurrency: r.integer(EnvPasswordHashConcurrency, DefaultPasswordHashConcurrency, 1, MaxPasswordHashConcurrency),
		Queue:       r.integer(EnvPasswordHashQueue, DefaultPasswordHashQueue, 0, MaxPasswordHashQueue),
		MaxWait:     r.duration(EnvPasswordHashMaxWait, DefaultPasswordHashMaxWait),
	}
	if cfg.MaxWait > MaxPasswordHashMaxWait {
		r.fail(EnvPasswordHashMaxWait, "must be at most %s; got %s", MaxPasswordHashMaxWait, cfg.MaxWait)
	}
	return cfg
}

// boolean reads "true" or "false" (default false).
func (r *reader) boolean(key string) bool {
	raw := strings.ToLower(r.str(key, "false"))
	if !slices.Contains([]string{"true", "false"}, raw) {
		r.fail(key, "must be true or false; got %q", raw)
		return false
	}
	return raw == "true"
}

// positiveFloat reads a number greater than zero and at most maximum.
func (r *reader) positiveFloat(key string, def, maximum float64) float64 {
	raw := r.str(key, "")
	if raw == "" {
		return def
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(f) || f <= 0 || f > maximum {
		r.fail(key, "must be a number greater than 0 and at most %s; got %q", fmt.Sprint(maximum), raw)
		return def
	}
	return f
}
