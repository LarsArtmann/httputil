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
