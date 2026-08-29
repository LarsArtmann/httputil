package httputil

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkGenerateTimeOrderedID measures the amortized cost of the time-
// ordered ID generator, including the periodic crypto/rand refill path.
func BenchmarkGenerateTimeOrderedID(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = generateTimeOrderedID()
	}
}

// BenchmarkGenerateTimeOrderedIDParallel measures contention on the shared
// random buffer and counter under concurrent generation.
func BenchmarkGenerateTimeOrderedIDParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = generateTimeOrderedID()
		}
	})
}

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
	b.ResetTimer()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status %d", rec.Code)
		}
	}
}

// BenchmarkKeyedRateLimiterConfigValidate measures config validation cost,
// which every KeyedRateLimiterMiddleware construction pays.
func BenchmarkKeyedRateLimiterConfigValidate(b *testing.B) {
	cfg := DefaultKeyedRateLimiterConfig()

	b.ReportAllocs()

	for b.Loop() {
		if err := cfg.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkServerConfigValidateWithTLS measures ServerConfig validation with
// a populated TLSConfig, the heaviest Validate path.
func BenchmarkServerConfigValidateWithTLS(b *testing.B) {
	cfg := DefaultServerConfig()
	cfg.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS13}

	b.ReportAllocs()

	for b.Loop() {
		if err := cfg.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkKeyedRateLimiterMiddleware measures the per-request middleware
// overhead with a hot key (cache-hit path through the limiter pool).
func BenchmarkKeyedRateLimiterMiddleware(b *testing.B) {
	cfg := DefaultKeyedRateLimiterConfig()
	cfg.Limit = 1_000_000_000 // effectively unlimited: this measures middleware overhead, not rejection
	cfg.Window = time.Minute
	cfg.KeyExtractor = KeyExtractorFromRemoteAddr()

	mw := KeyedRateLimiterMiddleware(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.Background())
	req.RemoteAddr = "203.0.113.7:1234"

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status %d", rec.Code)
		}
	}
}
