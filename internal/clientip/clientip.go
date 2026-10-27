// Package clientip determines the originating client address of a request.
package clientip

import (
	"net"
	"net/http"
	"strings"
)

// FromRequest returns the originating client address. Render (and most
// PaaS/reverse proxies) terminate the connection themselves, so RemoteAddr
// is always the proxy's own address and the real client IP has to come from
// a header.
//
// Render sits behind Cloudflare, which sets CF-Connecting-IP from the
// connection it actually received, overwriting any value the client sent, so
// it can't be spoofed. X-Forwarded-For can: Render appends to whatever the
// client sent instead of replacing it, so its first entry is client
// controlled. It is only used as a fallback when CF-Connecting-IP is absent
// (e.g. running locally).
func FromRequest(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.Split(xff, ",")[0]); ip != "" {
			return ip
		}
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}
