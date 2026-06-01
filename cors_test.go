package httputil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_DefaultConfig_Preflight(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodOptions, "/test", "http://example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	assertAllowOrigin(t, rec, "*")
}

func TestCORS_DefaultConfig_ActualRequest(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	middleware := CORS(cfg)

	called := false
	inner := newCountingHandler(&called)
	req := newTestRequest(http.MethodGet, "/test", "http://example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if !called {
		t.Error("next handler should be called for actual request")
	}

	assertAllowOrigin(t, rec, "*")
}

func TestCORS_SpecificOrigin(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "http://localhost:3000")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "http://localhost:3000")
}

func assertAllowOrigin(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != want {
		t.Errorf("Allow-Origin = %q, want %q", got, want)
	}
}

// serveWildcardCORS creates a CORS middleware with wildcard origin config and
// serves a request to the given URL, returning the response recorder.
func serveWildcardCORS(targetURL string) *httptest.ResponseRecorder {
	cfg := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", targetURL)
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	return rec
}

func TestCORSConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestCORSConfig_Validate_CredentialsWithAllOrigins(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for credentials+allorigins")
	}

	if !errors.Is(err, errCredentialsWithAllOrigins) {
		t.Errorf("Validate() error = %v, want errCredentialsWithAllOrigins", err)
	}
}

func TestCORSConfig_Validate_NegativeMaxAge(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
		MaxAge:         -1,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative MaxAge")
	}

	if !errors.Is(err, errNegativeMaxAge) {
		t.Errorf("Validate() error = %v, want errNegativeMaxAge", err)
	}
}

func TestCORS_CredentialsAndAllOrigins_InvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("config should be invalid: credentials+allorigins")
	}
}

func TestCORS_EmptyAllowedOrigins(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{},
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "*")
}

func TestCORS_OptionsPassthrough(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:     []string{"*"},
		AllowAllOrigins:    true,
		OptionsPassthrough: true,
	}
	middleware := CORS(cfg)

	called := false
	inner := newCountingHandler(&called)
	req := newTestRequest(http.MethodOptions, "/test", "http://example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler should be called with OptionsPassthrough")
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:  []string{"http://localhost:3000"},
		AllowAllOrigins: false,
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "*")
}

func TestCORS_MaxAgeZero(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins:  []string{"*"},
		AllowAllOrigins: true,
		MaxAge:          0,
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Max-Age"); got != "" {
		t.Errorf("Max-Age header = %q, want empty when MaxAge is 0", got)
	}
}

func BenchmarkCORS(b *testing.B) {
	cfg := DefaultCORSConfig()
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/test", "http://example.com")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func TestCORS_WildcardOriginMatch(t *testing.T) {
	t.Parallel()

	rec := serveWildcardCORS("http://sub.example.com")

	assertAllowOrigin(t, rec, "http://sub.example.com")
}

func TestCORS_WildcardOriginNoMatch(t *testing.T) {
	t.Parallel()

	rec := serveWildcardCORS("http://other.com")

	assertAllowOrigin(t, rec, "*")
}
