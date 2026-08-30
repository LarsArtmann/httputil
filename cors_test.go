package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

	assertStatus(t, rec, http.StatusNoContent)

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

func newCredentialsWithAllOriginsConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
	}
}

func TestCORSConfig_Validate_CredentialsWithAllOrigins(t *testing.T) {
	t.Parallel()

	cfg := newCredentialsWithAllOriginsConfig()

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

	cfg := newCredentialsWithAllOriginsConfig()

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
	req := newTestRequest(http.MethodGet, "/", "")
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

	assertHeader(t, rec, "Access-Control-Max-Age", "")
}

func BenchmarkCORS(b *testing.B) {
	cfg := DefaultCORSConfig()
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/test", "http://example.com")

	b.ReportAllocs()

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func TestCORS_WildcardOrigin(t *testing.T) {
	t.Parallel()

	assertAllowOrigin(t, serveWildcardCORS("http://sub.example.com"), "http://sub.example.com")
	assertAllowOrigin(t, serveWildcardCORS("http://other.com"), "*")
}

func FuzzCORS(f *testing.F) {
	f.Add("http://example.com")
	f.Add("https://sub.example.com")
	f.Add("http://other.com")
	f.Add("")
	f.Add("*")
	f.Add("not-a-url")
	f.Add("http://localhost:8080")

	cfg := CORSConfig{
		AllowedOrigins: []string{"http://example.com", "*.example.com"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
		AllowedHeaders: []string{"Content-Type"},
	}

	f.Fuzz(func(t *testing.T, origin string) {
		handler := CORS(cfg)(newNoOpHandler())

		req := newTestRequest(http.MethodGet, "/", origin)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		// Should not panic and should return a valid status.
		if rec.Code < 100 || rec.Code > 599 {
			t.Errorf("status = %d, not a valid HTTP status", rec.Code)
		}
	})
}

func TestCORS_ConcurrentRequests_NoRace(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"https://alpha.example.com", "https://beta.example.com"},
		AllowedMethods: []string{http.MethodGet},
		AllowedHeaders: []string{headerContentType},
	}
	handler := CORS(cfg)(newNoOpHandler())

	origins := []string{"https://alpha.example.com", "https://beta.example.com"}

	var waitGroup sync.WaitGroup

	for idx := range 100 {
		waitGroup.Add(1)

		go func(idx int) {
			defer waitGroup.Done()

			origin := origins[idx%len(origins)]
			req := newTestRequest(http.MethodGet, "/", "")
			req.Header.Set("Origin", origin)

			rec := newRecorder()

			handler.ServeHTTP(rec, req)

			got := rec.Header().Get("Access-Control-Allow-Origin")
			if got != origin {
				t.Errorf("origin = %q, got Allow-Origin = %q, want %q", origin, got, origin)
			}
		}(idx)
	}

	waitGroup.Wait()
}

// FuzzCORSWildcardPattern fuzzes matchWildcardOrigin with unusual and
// adversarial pattern/origin pairs (*.*, .*, a.*.b, empty, unicode, etc.).
// It enforces that the matcher never panics, is deterministic, and only ever
// returns true for "*."-prefixed patterns.
func FuzzCORSWildcardPattern(f *testing.F) {
	f.Add("*.example.com", "https://sub.example.com")
	f.Add("*.*", "http://a.b")
	f.Add(".*", "http://x")
	f.Add("a.*.b", "http://a.x.b")
	f.Add("*.", "http://.")
	f.Add("*..", "http://a..")
	f.Add("", "")
	f.Add("*", "http://anything")
	f.Add("*.example.com", "http://example.com")
	f.Add("*.exämple.com", "http://ö.exämple.com")

	f.Fuzz(func(t *testing.T, pattern, origin string) {
		got := matchWildcardOrigin(pattern, origin)

		// A pure matcher must be stable across calls.
		if matchWildcardOrigin(pattern, origin) != got {
			t.Fatalf(
				"matchWildcardOrigin not deterministic for pattern=%q origin=%q",
				pattern,
				origin,
			)
		}

		// Contract: only "*."-prefixed patterns can ever match.
		if !strings.HasPrefix(pattern, "*.") && got {
			t.Fatalf("matched non-wildcard pattern=%q origin=%q", pattern, origin)
		}
	})
}

// TestCORS_ExposedHeadersAndMaxAge covers the ExposedHeaders and MaxAge header
// branches that existing tests don't exercise.
func TestCORS_ExposedHeadersAndMaxAge(t *testing.T) {
	t.Parallel()

	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet},
		ExposedHeaders: []string{"X-Custom-Header"},
		MaxAge:         7200,
	}
	middleware := CORS(cfg)

	inner := newNoOpHandler()
	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	middleware(inner).ServeHTTP(rec, req)

	assertHeader(t, rec, "Access-Control-Expose-Headers", "X-Custom-Header")
	assertHeader(t, rec, "Access-Control-Max-Age", "7200")
}

// TestCORS_InvalidConfigLogsAndContinues verifies that an invalid CORS config
// (AllowCredentials + AllowAllOrigins) is logged via slog but does not prevent
// the middleware from constructing and serving requests.
func TestCORS_InvalidConfigLogsAndContinues(t *testing.T) {
	t.Parallel()

	// Browsers reject credentials with wildcard origins, so Validate flags
	// this combination. The constructor must log and still return a handler.
	cfg := CORSConfig{
		AllowAllOrigins:  true,
		AllowCredentials: true,
	}

	var called bool

	handler := CORS(cfg)(newCountingHandler(&called))
	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called (invalid config should log and continue)")
	}
}
