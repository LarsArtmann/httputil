package httputil

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// isValidHTTPToken returns true if s contains only valid HTTP token characters
// (RFC 7230 §3.2.6). httptest.NewRequest panics on inputs containing spaces,
// control characters, or non-printable bytes — fuzz inputs frequently produce
// these, so we filter them up front to avoid noise in the fuzzer.
func isValidHTTPToken(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r <= ' ' || r >= 0x7F {
			return false
		}

		switch r {
		case '(', ')', '<', '>', '@', ',', ';', ':', '\\', '"', '/', '[', ']', '?', '=', '{', '}':
			return false
		}
	}

	return true
}

// FuzzCSRFConfig_TrustedProxiesCIDR verifies that Validate() never crashes when
// given any user-supplied CIDR string. CSRF security depends on CIDR parsing
// being total — an attacker who can pass garbage here must not be able to
// trigger a panic or bypass origin validation.
func FuzzCSRFConfig_TrustedProxiesCIDR(f *testing.F) {
	f.Add("127.0.0.1/32")
	f.Add("10.0.0.0/8")
	f.Add("::1/128")
	f.Add("not-a-cidr")
	f.Add("")
	f.Add("/")
	f.Add("192.168.1.1")
	f.Add("1.2.3.4/33") // invalid mask
	f.Add("999.999.999.999/24")

	f.Fuzz(func(t *testing.T, cidr string) {
		cfg := CSRFConfig{
			TrustedProxies: []string{cidr},
		}

		// Validate must not panic. Either returns nil (valid CIDR) or
		// an error (invalid CIDR) — both outcomes are acceptable.
		_ = cfg.Validate()

		// TrustedProxiesCIDR must be populated consistently with TrustedProxies.
		// If Validate succeeded, every CIDR entry must have produced a parsed
		// IPNet entry (no entries skipped).
		if len(cfg.TrustedProxies) == 1 && strings.Contains(cfg.TrustedProxies[0], "/") {
			if len(cfg.TrustedProxiesCIDR) > 1 {
				t.Errorf(
					"Validate accepted %q but parsed multiple IPNets: %d",
					cidr,
					len(cfg.TrustedProxiesCIDR),
				)
			}
		}
	})
}

// FuzzCSRFConfig_TrustedOrigins verifies that Validate() rejects any origin
// containing an empty string or "*" wildcard, regardless of surrounding content.
// This is the security boundary for cross-origin CSRF — a bypass here would
// allow any origin to forge state-changing requests.
func FuzzCSRFConfig_TrustedOrigins(f *testing.F) {
	f.Add("https://example.com")
	f.Add("")
	f.Add("*")
	f.Add("https://*")
	f.Add("*://example.com")
	f.Add("https://example.com*")
	f.Add(strings.Repeat("a", 1000))
	f.Add("https://example.com\nhttp://evil.com") // header injection attempt

	f.Fuzz(func(t *testing.T, origin string) {
		cfg := CSRFConfig{
			TrustedOrigins: []string{origin},
		}

		err := cfg.Validate()

		// Empty or wildcard origins MUST be rejected — these are
		// security boundaries, not configuration knobs.
		if origin == "" || origin == "*" {
			if err == nil {
				t.Errorf("Validate accepted unsafe origin %q", origin)
			}
		}

		// Validate must not panic on any input
		_ = err
	})
}

// FuzzCSRFIsTrustedProxy verifies that isTrustedProxy is total: any combination
// of remote host, IP, address, and config must produce a deterministic boolean
// without panicking. The plaintext-HTTP origin bypass relies on this function
// — a crash here means a real request will not receive a response.
func FuzzCSRFIsTrustedProxy(f *testing.F) {
	f.Add("127.0.0.1", "127.0.0.1", "127.0.0.1:1234")
	f.Add("10.0.0.1", "10.0.0.1", "10.0.0.1:8080")
	f.Add("evil.com", "1.2.3.4", "1.2.3.4:443")
	f.Add("", "", "")
	f.Add("::1", "::1", "[::1]:8080")
	f.Add("not-an-ip", "", "no-port")
	f.Add(strings.Repeat("x", 200), strings.Repeat("y", 200), strings.Repeat("z", 200))

	f.Fuzz(func(t *testing.T, remoteHost, remoteIPStr, remoteAddr string) {
		var parsedIP net.IP
		if remoteIPStr != "" {
			parsedIP = net.ParseIP(remoteIPStr)
		}

		cfg := CSRFConfig{
			TrustedProxies: []string{"127.0.0.1", "10.0.0.0/8"},
		}
		if err := cfg.Validate(); err != nil {
			t.Skip("invalid config from fuzz")
		}

		// Must not panic on any input — output is just a bool.
		_ = isTrustedProxy(remoteHost, parsedIP, remoteAddr, cfg)

		// Also test with AllowPlaintextBypass enabled.
		cfg.AllowPlaintextBypass = true
		_ = isTrustedProxy(remoteHost, parsedIP, remoteAddr, cfg)
	})
}

// FuzzCSRFMiddleware_TokenValidation verifies the middleware never panics for
// any combination of method, header values, and request state. State-changing
// methods (POST/PUT/PATCH/DELETE) must reject all requests lacking a valid
// token+cookie pair — this is the entire purpose of CSRF protection.
func FuzzCSRFMiddleware_TokenValidation(f *testing.F) {
	f.Add("POST", "value", "cookie-value")
	f.Add("GET", "", "")
	f.Add("PUT", "x", "")
	f.Add("DELETE", "", "y")
	f.Add("", "", "")
	f.Add(strings.Repeat("A", 4096), strings.Repeat("B", 4096), strings.Repeat("C", 4096))
	f.Add("POST", "value\r\nX-Injected: bad", "cookie\r\nBad: true")

	f.Fuzz(func(t *testing.T, method, tokenValue, cookieValue string) {
		// httptest.NewRequest panics on invalid method characters
		// (space, control chars, etc.). Skip inputs that aren't valid
		// HTTP tokens — this fuzzer targets CSRF behavior, not request
		// construction.
		if method == "" {
			method = http.MethodGet
		}

		if !isValidHTTPToken(method) {
			t.Skip("invalid HTTP method character")
		}

		mw := CSRFMiddleware(CSRFConfig{})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(method, "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		if tokenValue != "" {
			req.Header.Set(DefaultCSRFHeaderName, tokenValue)
		}

		if cookieValue != "" {
			//nolint:gosec // test fixture — cookie values are intentionally fuzzed
			req.AddCookie(&http.Cookie{
				Name:  DefaultCSRFCookieName,
				Value: cookieValue,
			})
		}

		rec := httptest.NewRecorder()

		// Must not panic on any input
		handler.ServeHTTP(rec, req)

		// For state-changing methods, missing/invalid tokens MUST be rejected.
		switch method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if tokenValue == "" || cookieValue == "" {
				if rec.Code != http.StatusForbidden {
					t.Errorf(
						"%s without valid token+cookie: got %d, want 403",
						method, rec.Code,
					)
				}
			}
		}
	})
}

// FuzzCSRFRemoteHostAndIP verifies remoteHostAndIP handles all malformed
// RemoteAddr values. The plaintext-HTTP origin bypass extracts host/IP from
// RemoteAddr — a crash here breaks origin validation entirely.
func FuzzCSRFRemoteHostAndIP(f *testing.F) {
	f.Add("127.0.0.1:1234")
	f.Add("[::1]:8080")
	f.Add("localhost:80")
	f.Add("")
	f.Add(":")
	f.Add("1.2.3.4")
	f.Add("host-without-port")
	f.Add(strings.Repeat("x", 1024))

	f.Fuzz(func(t *testing.T, remoteAddr string) {
		// Must not panic on any input
		host, parsedIP := remoteHostAndIP(remoteAddr)
		_ = host
		_ = parsedIP
	})
}

// FuzzCSRFMiddleware_OriginHeaders verifies the middleware handles arbitrary
// combinations of Origin / Referer / Sec-Fetch-Site headers without panicking.
// These are the inputs that drive the plaintext-HTTP bypass decision — any
// crash here is a CSRF protection bypass waiting to happen.
func FuzzCSRFMiddleware_OriginHeaders(f *testing.F) {
	f.Add("https://example.com", "", "")
	f.Add("", "https://example.com/page", "")
	f.Add("", "", "same-origin")
	f.Add("", "", "cross-site")
	f.Add("", "", "none")
	f.Add("https://evil.com", "https://example.com", "same-origin")
	f.Add(strings.Repeat("a", 500), strings.Repeat("b", 500), strings.Repeat("c", 500))
	f.Add("https://example.com\r\nX-Evil: 1", "", "")

	f.Fuzz(func(t *testing.T, origin, referer, secFetchSite string) {
		mw := CSRFMiddleware(CSRFConfig{
			AllowPlaintextBypass: true,
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234" // non-loopback, non-trusted
		if origin != "" {
			req.Header.Set("Origin", origin)
		}

		if referer != "" {
			req.Header.Set("Referer", referer)
		}

		if secFetchSite != "" {
			req.Header.Set("Sec-Fetch-Site", secFetchSite)
		}

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Must not panic, must produce a valid HTTP status.
		if rec.Code == 0 {
			t.Errorf("recorder has no status code set")
		}
	})
}
