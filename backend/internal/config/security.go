package config

import (
	"net/netip"
	"strings"
	"time"
)

// Security environment variables (Phase 4 Group 6). Documented in backend/README.md and
// backend/.env.example.
const (
	// EnvTrustedProxies lists the reverse proxies or load balancers whose X-Forwarded-For header is
	// believed: comma-separated IP addresses or CIDR prefixes. Empty (the default): none, the direct
	// peer address is the client address.
	EnvTrustedProxies = "TRUSTED_PROXIES"
	// EnvSignInMaxAttempts and EnvSignInWindow set the per-source sign-in rate limit of the portal.
	EnvSignInMaxAttempts = "SIGNIN_RATE_LIMIT_MAX_ATTEMPTS"
	EnvSignInWindow      = "SIGNIN_RATE_LIMIT_WINDOW"
)

// Sign-in rate-limit defaults and bounds. The defaults are explained in backend/README.md (and in
// internal/signinlimit): at most 50 attempts per source address (IPv4) or /64 (IPv6) in 15 minutes.
const (
	DefaultSignInMaxAttempts = 50
	DefaultSignInWindow      = 15 * time.Minute
	MaxSignInMaxAttempts     = 100000
	MinSignInWindow          = time.Minute
	MaxSignInWindow          = 24 * time.Hour
	// maxTrustedProxies bounds the list (each request is checked against every entry).
	maxTrustedProxies = 64
)

// SignInLimitConfig is the per-source sign-in rate limit.
type SignInLimitConfig struct {
	MaxAttempts int
	Window      time.Duration
}

func (r *reader) trustedProxies() []netip.Prefix {
	raw := r.str(EnvTrustedProxies, "")
	if raw == "" {
		return nil
	}
	var out []netip.Prefix
	for item := range strings.SplitSeq(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
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
		// Every address would be trusted, so any client could choose its address.
		if p.Bits() == 0 {
			r.fail(EnvTrustedProxies, "must not contain %q, which trusts every address", item)
			continue
		}
		if p.Addr().Is4In6() {
			r.fail(EnvTrustedProxies, "%q: use the IPv4 form", item)
			continue
		}
		out = append(out, p)
	}
	if len(out) > maxTrustedProxies {
		r.fail(EnvTrustedProxies, "must list at most %d entries; got %d", maxTrustedProxies, len(out))
	}
	return out
}

func (r *reader) signInLimit() SignInLimitConfig {
	cfg := SignInLimitConfig{
		MaxAttempts: r.integer(EnvSignInMaxAttempts, DefaultSignInMaxAttempts, 1, MaxSignInMaxAttempts),
		Window:      r.duration(EnvSignInWindow, DefaultSignInWindow),
	}
	if cfg.Window != 0 && (cfg.Window < MinSignInWindow || cfg.Window > MaxSignInWindow) {
		r.fail(EnvSignInWindow, "must be from %s to %s; got %s", MinSignInWindow, MaxSignInWindow, cfg.Window)
	}
	return cfg
}
