package httputil

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// BenchmarkKeyedRateLimiter_MaxKeysChurn measures the true slow path under
// sustained MaxKeys pressure: every iteration inserts a fresh key, so each
// insert at capacity pays the heap-pop eviction plus push.
func BenchmarkKeyedRateLimiter_MaxKeysChurn(b *testing.B) {
	const maxKeys = 1024

	p := newPerKeyLimiter(rate.Limit(1e9), 1_000_000_000, nil, "0", time.Hour, maxKeys)

	i := 0

	for b.Loop() {
		i++
		_ = p.limiter("key-" + strconv.Itoa(i))
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

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:1234"

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status %d", rec.Code)
		}
	}
}
