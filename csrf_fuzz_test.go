package httputil

import (
	"context"
	"encoding/json/v2"
	"html"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		err := cfg.Validate()

		// Parsing lives in withParsedTrustedProxies (Validate is pure). If
		// Validate accepted a single CIDR entry, the parse must produce
		// exactly one IPNet; an invalid entry must produce none.
		parsed := cfg.withParsedTrustedProxies()

		if err == nil && strings.Contains(cidr, "/") && len(parsed.TrustedProxiesCIDR) != 1 {
			t.Errorf(
				"Validate accepted %q but parsed %d IPNets, want 1",
				cidr,
				len(parsed.TrustedProxiesCIDR),
			)
		}

		if err != nil && len(parsed.TrustedProxiesCIDR) != 0 {
			t.Errorf(
				"Validate rejected %q but parse produced %d IPNets, want 0",
				cidr,
				len(parsed.TrustedProxiesCIDR),
			)
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
	f.Add("example.com") // scheme-less: parses, must fail the shape gate
	f.Add("https://")    // host-less: parses, must fail the shape gate
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

		// Entries that cannot parse or lack scheme/host MUST be rejected:
		// an unparseable entry makes nosurf.StaticOrigins fail wholesale,
		// and a scheme-less entry can never match an Origin header — both
		// silently change trust semantics.
		originURL, parseErr := parseTrustedOrigin(origin)
		if (parseErr != nil || originURL.Scheme == "" || originURL.Host == "") && err == nil {
			t.Errorf("Validate accepted non-scheme://host origin %q", origin)
		}
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

		// For state-changing methods, every fuzzed input is an invalid
		// token+cookie pair (missing, empty, mismatched, or not a real
		// masked token) and MUST be rejected with the CSRF rejection —
		// including the double-submit bypass class where both values are
		// present but differ. Safe methods must never be rejected here.
		unsafe := true

		switch method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			unsafe = false
		}

		if !unsafe {
			return
		}

		if rec.Code != http.StatusForbidden {
			t.Errorf(
				"%s with bogus token=%q cookie=%q: got %d, want 403",
				method, tokenValue, cookieValue, rec.Code,
			)

			return
		}

		if body := rec.Body.String(); !strings.Contains(body, "csrf_invalid") {
			t.Errorf(
				"%s with bogus token: body = %q, want the csrf_invalid rejection",
				method, body,
			)
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
// combinations of method / Origin / Referer / Sec-Fetch-Site headers without
// panicking. These are the inputs that drive the plaintext-HTTP bypass and
// attestation-conflict decisions — any crash here is a CSRF protection bypass
// waiting to happen. The oracle additionally pins the attestation-conflict
// contract: an unsafe method with a literal same-origin attestation and an
// Origin that is neither absent, null, nor self MUST be rejected with 403
// (ErrCSRFAttestationConflict) — the forged-attestation defense applies to
// every unsafe method, not just GET.
func FuzzCSRFMiddleware_OriginHeaders(f *testing.F) {
	f.Add(http.MethodGet, "https://example.com", "", "")
	f.Add(http.MethodGet, "", "https://example.com/page", "")
	f.Add(http.MethodGet, "", "", "same-origin")
	f.Add(http.MethodGet, "", "", "cross-site")
	f.Add(http.MethodGet, "", "", "none")
	f.Add(http.MethodGet, "https://evil.com", "https://example.com", "same-origin")
	f.Add(
		http.MethodPost,
		strings.Repeat("a", 500),
		strings.Repeat("b", 500),
		strings.Repeat("c", 500),
	)
	f.Add(http.MethodGet, "https://example.com\r\nX-Evil: 1", "", "")
	// Contradictory combos on unsafe methods: the test request is plain HTTP
	// with Host example.com, so an https Origin contradicts the attestation.
	f.Add(http.MethodPost, "https://example.com", "", "same-origin")
	f.Add(http.MethodPost, "https://evil.com", "", "same-origin")
	f.Add(http.MethodPut, "https://evil.com", "", "same-origin")
	f.Add(http.MethodPatch, "https://evil.com", "", "same-origin")
	f.Add(http.MethodDelete, "https://evil.com", "", "same-origin")
	f.Add(http.MethodPost, "not-an-origin", "", "same-origin")
	f.Add(http.MethodPost, "https://example.com\r\nX-Evil: 1", "", "same-origin")
	// Boundary seeds: attestation with absent, null, or self origins is
	// consistent and must NOT be treated as a conflict.
	f.Add(http.MethodPost, "", "", "same-origin")
	f.Add(http.MethodPost, "null", "", "same-origin")
	f.Add(http.MethodPost, "http://example.com", "", "same-origin")
	f.Add(http.MethodGet, "null", "", "same-origin")

	f.Fuzz(func(t *testing.T, method, origin, referer, secFetchSite string) {
		// httptest.NewRequest panics on invalid method characters. Skip
		// inputs that aren't valid HTTP tokens — this fuzzer targets CSRF
		// behavior, not request construction.
		if method == "" {
			method = http.MethodGet
		}

		if !isValidHTTPToken(method) {
			t.Skip("invalid HTTP method character")
		}

		mw := CSRFMiddleware(CSRFConfig{
			AllowPlaintextBypass: true,
		})

		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(method, "/", nil)
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

		// Attestation-conflict oracle, computed independently of the
		// middleware internals: unsafe method (the RFC safe set is
		// GET/HEAD/OPTIONS/TRACE) + literal same-origin attestation + an
		// Origin that is non-empty, not "null", and either unparseable or
		// not the request's own http://example.com origin. No trusted
		// origins are configured, so none can excuse a mismatch.
		unsafe := true
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			unsafe = false
		}

		wantConflict := unsafe && secFetchSite == "same-origin" &&
			origin != "" && origin != "null"

		if wantConflict {
			// Host comparison is case-insensitive (DNS), and no trusted
			// proxies are configured here, so the scheme view is the local
			// connection's (plain HTTP).
			if parsed, parseErr := url.Parse(origin); parseErr == nil &&
				strings.EqualFold(parsed.Host, req.Host) && parsed.Scheme == "http" {
				wantConflict = false
			}
		}

		if wantConflict {
			// The 403 must be the attestation-conflict rejection, not a
			// nosurf token failure wearing the same status: the status and
			// the body both name it. Asserting the body keeps the oracle
			// sensitive to the attestation check being deleted (nosurf
			// would still 403 on the missing token, but with csrf_invalid).
			if rec.Code != http.StatusForbidden {
				t.Errorf(
					"contradicted attestation (%s, Sec-Fetch-Site: same-origin, Origin %q): status = %d, want %d",
					method,
					origin,
					rec.Code,
					http.StatusForbidden,
				)
			}

			if body := rec.Body.String(); !strings.Contains(
				body,
				"csrf.origin_attestation_conflict",
			) {
				t.Errorf(
					"contradicted attestation (%s, Origin %q): body = %q, want csrf.origin_attestation_conflict rejection",
					method,
					origin,
					body,
				)
			}
		}

		if !wantConflict && rec.Code == 0 {
			t.Errorf("consistent request (%s, Origin %q): no status set", method, origin)
		}
	})
}

// FuzzCSRFTokenHTMLFormatters verifies the template-formatting helpers
// (hx-headers attribute, HTML meta tag, hidden form field) render any context
// token without crashing, and pin the token-less contract: with no token in
// context every helper returns "" (the path every template hits when no CSRF
// middleware ran upstream). The hx-headers payload must round-trip: HTML-
// unescaping the attribute body yields JSON carrying the exact token, so a
// quote-bearing token can never terminate the attribute early.
func FuzzCSRFTokenHTMLFormatters(f *testing.F) {
	f.Add("")
	f.Add("masked-token-value")
	f.Add(`"quoted" & <escaped>`)
	f.Add("tok\r\nInjected: 1")
	f.Add("</script><script>alert(1)</script>")
	f.Add(strings.Repeat("x", 4096))

	f.Fuzz(func(t *testing.T, token string) {
		req := httptest.NewRequest(http.MethodGet, "/", nil).
			WithContext(WithCSRFToken(context.Background(), token))

		hx := CSRFTokenHXHeaders(req)
		meta := CSRFTokenHTMLMeta(req)
		field := CSRFTokenFormField(req)

		if token == "" {
			if hx != "" || meta != "" || field != "" {
				t.Errorf(
					"empty token renders hx=%q meta=%q field=%q, want all empty",
					hx, meta, field,
				)
			}

			return
		}

		escaped := html.EscapeString(token)

		if !strings.Contains(meta, escaped) {
			t.Errorf("meta tag %q does not contain the escaped token %q", meta, escaped)
		}

		if !strings.Contains(field, escaped) {
			t.Errorf("form field %q does not contain the escaped token %q", field, escaped)
		}

		const hxPrefix = `hx-headers='`

		if !strings.HasPrefix(hx, hxPrefix) || !strings.HasSuffix(hx, `'`) {
			t.Errorf("hx-headers output %q is not a single-quoted attribute", hx)

			return
		}

		body := html.UnescapeString(strings.TrimSuffix(strings.TrimPrefix(hx, hxPrefix), `'`))

		var decoded map[string]string
		if err := json.Unmarshal([]byte(body), &decoded); err != nil {
			t.Errorf("hx-headers attribute body %q does not decode as JSON: %v", body, err)

			return
		}

		if got := decoded[DefaultCSRFHeaderName]; got != token {
			t.Errorf("hx-headers round-trip token = %q, want %q", got, token)
		}
	})
}
