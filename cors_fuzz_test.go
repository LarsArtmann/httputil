package httputil

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// corsAllowlistHasWildcardEntry reports whether any allowlist entry uses
// wildcard matching ("*" or a "*.domain" prefix pattern), which makes the
// exact-echo oracle inapplicable.
func corsAllowlistHasWildcardEntry(allowlist []string) bool {
	for _, entry := range allowlist {
		if entry == "*" || strings.HasPrefix(entry, "*.") {
			return true
		}
	}

	return false
}

// FuzzCORSOriginEcho fuzzes arbitrary request origins against arbitrary
// allowlists and pins the echo property: the Access-Control-Allow-Origin
// response value is always "", "*", or a byte-exact echo of the request
// origin, never a third-party origin. For exact-string allowlists with
// DenyUnmatched, the stronger oracle holds: allowlisted origins are echoed
// exactly and unlisted origins get no header at all.
func FuzzCORSOriginEcho(f *testing.F) {
	f.Add("https://example.com", "https://example.com", "https://other.com", false, true, false)
	f.Add("https://sub.example.com", "*.example.com", "https://example.com", false, true, false)
	f.Add("https://evil.com", "https://good.com", "https://also-good.com", false, true, false)
	f.Add("https://any.com", "https://ignored.com", "", true, false, false)
	f.Add("", "https://a.com", "", false, false, false)
	f.Add("null", "null", "null", false, false, true)
	f.Add("HTTP://Case.Com", "http://case.com", "", false, true, false)

	f.Fuzz(func(
		t *testing.T,
		origin, allowA, allowB string,
		allowAll, denyUnmatched, allowCredentials bool,
	) {
		if allowAll && allowCredentials {
			t.Skip("AllowCredentials with AllowAllOrigins is rejected by Validate")
		}

		allowlist := []string{allowA, allowB}

		cfg := CORSConfig{
			AllowedOrigins:     allowlist,
			AllowedMethods:     []string{http.MethodGet},
			AllowedHeaders:     []string{},
			ExposedHeaders:     []string{},
			AllowCredentials:   allowCredentials,
			MaxAge:             0,
			AllowAllOrigins:    allowAll,
			OptionsPassthrough: false,
			DenyUnmatched:      denyUnmatched,
		}

		handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", origin)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		got := rec.Header().Get("Access-Control-Allow-Origin")

		if got != "" && got != "*" && got != origin {
			t.Errorf(
				"reflected third-party origin %q for request origin %q (allowlist %q)",
				got,
				origin,
				allowlist,
			)
		}

		if allowAll && got != "*" {
			t.Errorf("AllowAllOrigins ACAO = %q, want %q", got, "*")
		}

		exactOnly := !allowAll && denyUnmatched && origin != "" &&
			!corsAllowlistHasWildcardEntry(allowlist)

		if !exactOnly {
			return
		}

		if slices.Contains(allowlist, origin) {
			if got != origin {
				t.Errorf("allowlisted origin %q got ACAO %q, want exact echo", origin, got)
			}

			return
		}

		if got != "" {
			t.Errorf("unlisted origin %q got ACAO %q, want no header", origin, got)
		}
	})
}
