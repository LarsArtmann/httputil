package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkCSRFMiddleware_PlainHTTPNosurf measures the full nosurf token
// issuance path for a plain (non-TLS, non-localhost) HTTP request, which is
// the worst-case configuration: nosurf cannot mark the request secure and
// performs the complete cookie dance on every request.
func BenchmarkCSRFMiddleware_PlainHTTPNosurf(b *testing.B) {
	cfg := CSRFConfig{Secure: true, SameSite: http.SameSiteLaxMode}
	mw := CSRFMiddleware(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status %d", rec.Code)
		}
	}
}

// BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck measures the unsafe-method
// hot path, where the attestation-consistency check runs on every request before
// nosurf: a POST carrying a consistent same-origin attestation continues into the
// full nosurf rejection, so the row captures middleware overhead on
// state-changing requests rather than the token-issuance GET path.
func BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck(b *testing.B) {
	cfg := CSRFConfig{Secure: true, SameSite: http.SameSiteLaxMode}
	mw := CSRFMiddleware(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			b.Fatalf("unexpected status %d", rec.Code)
		}
	}
}
