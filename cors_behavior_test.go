package httputil

import (
	"net/http"
	"testing"
)

// These tests specify observable CORS behaviors — what a consumer sees, not how
// it is implemented. They document intentionally surprising behaviors (notably the
// allowlist fallback) so that any future change to them is a deliberate decision.

// TestCORS_BareLiteralFallsBackToWildcardForUnmatchedOrigin documents the
// zero-value behavior of a bare CORSConfig literal: when DenyUnmatched is left
// at its zero value (false) and AllowAllOrigins is false, a request origin that
// matches no entry in AllowedOrigins still responds with
// Access-Control-Allow-Origin: *. Note that DefaultCORSConfig() sets
// DenyUnmatched: true, so this wildcard fallback only applies to literals
// constructed without the default helper. Consumers who want unmatched origins
// denied should use DefaultCORSConfig() or set DenyUnmatched: true explicitly
// (see TestCORS_DenyUnmatchedSuppressesHeader).
func TestCORS_BareLiteralFallsBackToWildcardForUnmatchedOrigin(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"https://allowed.example.com"},
		AllowedMethods: []string{http.MethodGet},
		AllowedHeaders: []string{"Content-Type"},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "https://attacker.evil")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	// Zero-value behavior: bare literal with DenyUnmatched=false still yields "*".
	assertAllowOrigin(t, rec, "*")
}

// TestCORS_DenyUnmatchedSuppressesHeader specifies that when DenyUnmatched is
// true, an origin matching no entry in AllowedOrigins produces no
// Access-Control-Allow-Origin header at all — the allowlist is enforced.
func TestCORS_DenyUnmatchedSuppressesHeader(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"https://allowed.example.com"},
		AllowedMethods: []string{http.MethodGet},
		AllowedHeaders: []string{"Content-Type"},
		DenyUnmatched:  true,
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "https://attacker.evil")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf(
			"Allow-Origin = %q, want absent (DenyUnmatched=true)",
			got,
		)
	}
}

// assertCORSForOrigin runs the CORS middleware with the given allowed origins
// and request origin, then asserts that Access-Control-Allow-Origin matches
// wantOrigin. If wantOrigin is empty, asserts the header is absent (used when
// DenyUnmatched is enabled).
func assertCORSForOrigin(t *testing.T, allowedOrigins []string, requestOrigin, wantOrigin string) {
	t.Helper()

	cfg := CORSConfig{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodGet},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", requestOrigin)
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if wantOrigin == "" {
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q, want absent", got)
		}

		return
	}

	assertAllowOrigin(t, rec, wantOrigin)
}

// TestCORS_NoOriginHeaderDefaultsToWildcard documents that when no Origin header is
// present and AllowAllOrigins is false, the default wildcard is used.
func TestCORS_NoOriginHeaderDefaultsToWildcard(t *testing.T) {
	t.Parallel()

	assertCORSForOrigin(t, []string{"https://allowed.example.com"}, "", "*")
}

// TestCORS_WildcardPatternMatchesSubdomain specifies that "*.example.com" matches a
// real subdomain origin.
func TestCORS_WildcardPatternMatchesSubdomain(t *testing.T) {
	t.Parallel()

	assertCORSForOrigin(
		t,
		[]string{"*.example.com"},
		"https://api.example.com",
		"https://api.example.com",
	)
}

// TestCORS_WildcardPatternRejectsLookalikeDomain is a security edge case: "*.example.com"
// must NOT match "https://notexample.com" (no label boundary before "example").
func TestCORS_WildcardPatternRejectsLookalikeDomain(t *testing.T) {
	t.Parallel()

	// Lookalike does not match the wildcard, so the default fallback applies.
	assertCORSForOrigin(t, []string{"*.example.com"}, "https://notexample.com", "*")
}

// TestCORS_WildcardPatternRejectsLookalikeWithDenyUnmatched verifies that the
// lookalike domain is also denied the header when DenyUnmatched is set.
func TestCORS_WildcardPatternRejectsLookalikeWithDenyUnmatched(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
		AllowedMethods: []string{http.MethodGet},
		DenyUnmatched:  true,
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "https://notexample.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf(
			"Allow-Origin = %q, want absent (DenyUnmatched=true for lookalike)",
			got,
		)
	}
}

// TestCORS_WildcardPatternMatchesNestedSubdomain specifies that "*.example.com" also
// matches deeper subdomains like "a.b.example.com".
func TestCORS_WildcardPatternMatchesNestedSubdomain(t *testing.T) {
	t.Parallel()

	assertCORSForOrigin(
		t,
		[]string{"*.example.com"},
		"https://a.b.example.com",
		"https://a.b.example.com",
	)
}

// TestCORS_WildcardRejectsLookalikeWithPort ensures that a wildcard pattern
// does not match an origin that merely contains the domain as a substring in
// the port section.
func TestCORS_WildcardRejectsLookalikeWithPort(t *testing.T) {
	t.Parallel()

	assertCORSForOrigin(
		t,
		[]string{"*.example.com"},
		"https://evil.com:example.com",
		"*",
	)
}

// TestCORS_EmptyAllowedOriginsWithDenyUnmatched ensures that when the allowlist
// is empty and DenyUnmatched is true, all origins are denied.
func TestCORS_EmptyAllowedOriginsWithDenyUnmatched(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{},
		AllowedMethods: []string{http.MethodGet},
		DenyUnmatched:  true,
	}

	middleware := CORS(cfg)
	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "https://anywhere.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want absent (empty allowlist + DenyUnmatched)", got)
	}
}

func FuzzCORSOriginMatching(f *testing.F) {
	f.Add("https://api.example.com")
	f.Add("https://notexample.com")
	f.Add("https://evil.com:example.com")
	f.Add("")
	f.Add("*")

	f.Fuzz(func(t *testing.T, origin string) {
		t.Parallel()

		cfg := CORSConfig{
			AllowedOrigins: []string{"*.example.com", "https://allowed.example.com"},
			AllowedMethods: []string{http.MethodGet},
			DenyUnmatched:  true,
		}

		middleware := CORS(cfg)
		inner := newNoOpHandler()
		req := newTestRequest(http.MethodGet, "/", origin)
		rec := newRecorder()

		middleware(inner).ServeHTTP(rec, req)

		// With DenyUnmatched the response must either deny (absent header)
		// or echo the requested origin; a wildcard leak would defeat the
		// deny-unmatched contract pinned by TestCORS_DenyUnmatchedSuppressesHeader.
		// A request with no Origin header is not a CORS request (browsers do
		// not enforce Access-Control-Allow-Origin on it), so its value is
		// unconstrained.
		got := rec.Header().Get("Access-Control-Allow-Origin")

		if origin != "" && got != "" && got != origin {
			t.Errorf(
				"unexpected Allow-Origin = %q for origin %q under DenyUnmatched (empty or echo only)",
				got,
				origin,
			)
		}
	})
}
