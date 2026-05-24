package httputil

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP extracts the client IP address from the request using the following
// precedence: X-Forwarded-For (first entry), X-Real-IP, then RemoteAddr with
// net.SplitHostPort.
//
// Warning: trusts X-Forwarded-For and X-Real-IP headers without validation.
// Only use behind a reverse proxy that strips/overwrites these headers.
func ClientIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}

	return host
}
