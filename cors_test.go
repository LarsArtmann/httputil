package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_DefaultConfig_Preflight(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	middleware := CORS(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for preflight")
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()

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
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()

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

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rec := httptest.NewRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "http://localhost:3000")
}

func assertAllowOrigin(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != want {
		t.Errorf("Allow-Origin = %q, want %q", got, want)
	}
}
