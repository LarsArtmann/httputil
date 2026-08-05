package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BenchmarkCSRFMiddleware_GET measures the cost of CSRF middleware for GET
// requests — this is the common path (no validation required, just cookie
// generation). The benchmark exercises the full middleware stack including
// token generation, cookie setting, and context storage.
func BenchmarkCSRFMiddleware_GET(b *testing.B) {
	mw := CSRFMiddleware(CSRFConfig{})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkCSRFMiddleware_POSTWithToken measures the cost of CSRF middleware
// for state-changing requests that pass validation. This is the path through
// nosurf's token comparison, hash unmasking, and origin checking.
func BenchmarkCSRFMiddleware_POSTWithToken(b *testing.B) {
	mw := CSRFMiddleware(CSRFConfig{})

	// Establish a session: GET to obtain token + cookie.
	token, cookie := CSRFTestToken(mw)
	if token == "" || cookie == nil {
		b.Fatal("CSRFTestToken failed to establish session")
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Re-use the same token + cookie for all iterations — represents
	// a valid request flow.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set(DefaultCSRFHeaderName, token)
	req.AddCookie(cookie)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkCSRFMiddleware_POSTRejection measures the cost of CSRF middleware
// for state-changing requests that fail validation. This includes the
// failure handler invocation and the nosurf rejection path.
func BenchmarkCSRFMiddleware_POSTRejection(b *testing.B) {
	mw := CSRFMiddleware(CSRFConfig{})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Origin", "http://malicious.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkCSRFMiddleware_PostForm measures the cost of CSRF middleware when
// the token is supplied via a form field rather than the header. This is the
// path that goes through TranslateCSRFHeaders → PostFormValue lookup.
func BenchmarkCSRFMiddleware_PostForm(b *testing.B) {
	mw := CSRFMiddleware(CSRFConfig{})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := "csrf_token=test-token"

	b.ReportAllocs()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "127.0.0.1:1234"

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkCSRFConfig_Validate measures the cost of CSRFConfig.Validate().
// Called once at middleware construction time, but a benchmark gives a
// baseline for config-validation overhead.
func BenchmarkCSRFConfig_Validate(b *testing.B) {
	cfg := CSRFConfig{
		TrustedProxies: []string{"10.0.0.0/8", "192.168.0.0/16", "127.0.0.1/32"},
		TrustedOrigins: []string{
			"https://app.example.com",
			"https://admin.example.com",
		},
	}

	b.ReportAllocs()

	for b.Loop() {
		// Reset CIDR slice so each iteration mirrors construction-time validation.
		cfgCopy := cfg
		cfgCopy.TrustedProxiesCIDR = nil
		_ = cfgCopy.Validate()
	}
}

// BenchmarkCSRFTokenFromContext measures the cost of retrieving the CSRF
// token from request context. Called on every request via CSRFResponseHeaderMiddleware
// and template helpers.
func BenchmarkCSRFTokenFromContext(b *testing.B) {
	mw := CSRFMiddleware(CSRFConfig{})

	var captured *http.Request

	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if captured == nil {
		b.Fatal("handler did not capture request")
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = CSRFTokenFromContext(captured.Context())
	}
}
