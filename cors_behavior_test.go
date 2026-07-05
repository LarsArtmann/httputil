package httputil

import (
	"net/http"
	"testing"
)

// These tests specify observable CORS behaviors — what a consumer sees, not how
// it is implemented. They document intentionally surprising behaviors (notably the
// allowlist fallback) so that any future change to them is a deliberate decision.

// TestCORS_AllowlistFallsBackToWildcardForUnmatchedOriginByDefault documents a
// security-relevant default: when AllowAllOrigins is false and a request origin
// matches no entry in AllowedOrigins, the middleware still responds with
// Access-Control-Allow-Origin: *. Consumers who want unmatched origins denied
// must set DenyUnmatched: true (see TestCORS_DenyUnmatchedSuppressesHeader).
func TestCORS_AllowlistFallsBackToWildcardForUnmatchedOriginByDefault(t *testing.T) {
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

	// Default behavior: unmatched origin still gets "*".
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

// TestCORS_NoOriginHeaderDefaultsToWildcard documents that when no Origin header is
// present and AllowAllOrigins is false, the default wildcard is used.
func TestCORS_NoOriginHeaderDefaultsToWildcard(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"https://allowed.example.com"},
		AllowedMethods: []string{http.MethodGet},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "*")
}

// TestCORS_WildcardPatternMatchesSubdomain specifies that "*.example.com" matches a
// real subdomain origin.
func TestCORS_WildcardPatternMatchesSubdomain(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
		AllowedMethods: []string{http.MethodGet},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "https://api.example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "https://api.example.com")
}

// TestCORS_WildcardPatternRejectsLookalikeDomain is a security edge case: "*.example.com"
// must NOT match "https://notexample.com" (no label boundary before "example").
func TestCORS_WildcardPatternRejectsLookalikeDomain(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
		AllowedMethods: []string{http.MethodGet},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "https://notexample.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	// Lookalike does not match the wildcard, so the default fallback applies.
	assertAllowOrigin(t, rec, "*")
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

	cfg := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
		AllowedMethods: []string{http.MethodGet},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "https://a.b.example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "https://a.b.example.com")
}
