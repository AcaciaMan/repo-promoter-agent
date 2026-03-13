package ratelimit

import (
	"net"
	"net/http"
	"strings"
)

// ClientKeyFromRequest extracts a client identifier from the request,
// suitable for use as a rate-limiting key.
func ClientKeyFromRequest(r *http.Request) string {
	// 1. X-Forwarded-For: take the leftmost (first) IP.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for _, part := range strings.Split(xff, ",") {
			ip := strings.TrimSpace(part)
			if ip != "" {
				return stripPort(ip)
			}
		}
	}

	// 2. X-Real-IP
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return stripPort(xri)
	}

	// 3. RemoteAddr fallback
	return stripPort(r.RemoteAddr)
}

// stripPort removes the port from an address string.
// If net.SplitHostPort fails (e.g., no port present), the value is returned as-is.
func stripPort(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
