package middleware

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

type clientIPKey struct{}

// maxForwardedHops bounds how many X-Forwarded-For entries are examined, so a long header cannot
// cost unbounded work.
const maxForwardedHops = 32

// ClientIP determines the client IP address of every request and stores it in the request context
// (ClientIPFromContext). It never trusts a client-supplied header by default:
//
//   - The address is the direct peer of the connection (http.Request.RemoteAddr).
//   - Only when that peer is a configured trusted proxy is X-Forwarded-For used: its entries are read
//     from right to left (the most recent proxy appends last), trusted proxies are skipped, and the
//     first address that is not a trusted proxy is the client. If every entry is a trusted proxy, the
//     leftmost one is used. A malformed entry ends the search at the last address known to be valid,
//     so a header that cannot be parsed never replaces a trusted peer with a spoofed value.
//   - Other headers (Forwarded, X-Real-IP, ...) are ignored.
//
// With no trusted proxies (the default), X-Forwarded-For is never read. Behind a load balancer or
// reverse proxy, configure its addresses (TRUSTED_PROXIES), or every client appears to come from
// the proxy. IPv4-mapped IPv6 addresses are reported as IPv4. An invalid netip.Addr is stored when
// the peer address cannot be parsed.
func ClientIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	trusted = slices.Clone(trusted)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveClientIP(r, trusted)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}
}

// ClientIPFromContext returns the address stored by ClientIP (invalid when there is none).
func ClientIPFromContext(ctx context.Context) netip.Addr {
	ip, _ := ctx.Value(clientIPKey{}).(netip.Addr)
	return ip
}

func resolveClientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	peer := parseAddr(r.RemoteAddr, true)
	if !peer.IsValid() || !isTrusted(peer, trusted) {
		return peer
	}
	var hops []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(v, ",")...)
	}
	client := peer
	for i, seen := len(hops)-1, 0; i >= 0 && seen < maxForwardedHops; i, seen = i-1, seen+1 {
		hop := parseAddr(strings.TrimSpace(hops[i]), false)
		if !hop.IsValid() {
			return client // malformed: stop at the last address a trusted proxy vouched for
		}
		client = hop
		if !isTrusted(hop, trusted) {
			return hop
		}
	}
	return client
}

func isTrusted(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// parseAddr parses an IP address, or (withPort) an address with a port as in RemoteAddr. Zones are
// removed and IPv4-mapped IPv6 addresses are unmapped.
func parseAddr(s string, withPort bool) netip.Addr {
	if withPort {
		if host, _, err := net.SplitHostPort(s); err == nil {
			s = host
		}
	}
	s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
	ip, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}
	}
	return ip.Unmap().WithZone("")
}
