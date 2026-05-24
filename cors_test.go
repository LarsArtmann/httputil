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

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()

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
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()

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

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()

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

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()

	middleware(inner).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Max-Age"); got != "" {
		t.Errorf("Max-Age header = %q, want empty when MaxAge is 0", got)
	}
}

func BenchmarkCORS(b *testing.B) {
	cfg := DefaultCORSConfig()
	middleware := CORS(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := middleware(inner)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func TestCORS_WildcardOriginMatch(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
	}
	middleware := CORS(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://sub.example.com")

	rec := httptest.NewRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "http://sub.example.com")
}

func TestCORS_WildcardOriginNoMatch(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"*.example.com"},
	}
	middleware := CORS(cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://other.com")

	rec := httptest.NewRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertAllowOrigin(t, rec, "*")
}
