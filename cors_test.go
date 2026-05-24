package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_DefaultConfig_Preflight(t *testing.T) {
	t.Parallel()
	cfg := DefaultCORSConfig()
	mw := CORS(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for preflight")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	mw(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q, want %q", got, "*")
	}
}

func TestCORS_DefaultConfig_ActualRequest(t *testing.T) {
	t.Parallel()
	cfg := DefaultCORSConfig()
	mw := CORS(cfg)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	mw(inner).ServeHTTP(rec, req)

	if !called {
		t.Error("next handler should be called for actual request")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q, want %q", got, "*")
	}
}

func TestCORS_SpecificOrigin(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}
	mw := CORS(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	mw(inner).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
}
