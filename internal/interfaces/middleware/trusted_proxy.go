package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// TrustedProxy only honors client-IP headers when the direct peer belongs to
// an explicitly configured proxy network. Untrusted clients cannot rotate XFF
// values to evade rate limiting or forge audit addresses.
func TrustedProxy(trusted []netip.Prefix) func(http.Handler) http.Handler {
	isTrusted := func(addr netip.Addr) bool {
		for _, prefix := range trusted {
			if prefix.Contains(addr.Unmap()) {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer, ok := remoteIP(r.RemoteAddr)
			if !ok || !isTrusted(peer) {
				next.ServeHTTP(w, r)
				return
			}
			candidate, ok := forwardedClient(r, isTrusted)
			if ok {
				r.RemoteAddr = net.JoinHostPort(candidate.String(), "0")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func remoteIP(value string) (netip.Addr, bool) {
	if addrPort, err := netip.ParseAddrPort(value); err == nil {
		return addrPort.Addr().Unmap(), true
	}
	addr, err := netip.ParseAddr(strings.Trim(value, "[]"))
	return addr.Unmap(), err == nil
}

func forwardedClient(r *http.Request, isTrusted func(netip.Addr) bool) (netip.Addr, bool) {
	if raw := r.Header.Get("X-Forwarded-For"); raw != "" {
		parts := strings.Split(raw, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			addr, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
			if err != nil {
				return netip.Addr{}, false
			}
			addr = addr.Unmap()
			if !isTrusted(addr) {
				return addr, true
			}
		}
	}
	if raw := strings.TrimSpace(r.Header.Get("X-Real-IP")); raw != "" {
		addr, err := netip.ParseAddr(raw)
		return addr.Unmap(), err == nil
	}
	return netip.Addr{}, false
}
