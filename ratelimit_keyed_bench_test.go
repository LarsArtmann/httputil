package httputil

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// BenchmarkKeyedRateLimiter_Allow measures throughput of the per-key token
// bucket on the allow path with a small number of keys (low contention).
// This represents the common case: a small set of clients hitting the API
// within their rate limit.
func BenchmarkKeyedRateLimiter_Allow(b *testing.B) {
	cfg := KeyedRateLimiterConfig{
		Limit:        1_000_000,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	keys := []string{
		"10.0.0.1:1234",
		"10.0.0.2:1234",
		"10.0.0.3:1234",
		"10.0.0.4:1234",
		"10.0.0.5:1234",
	}

	requests := make([]*http.Request, len(keys))

	for i, addr := range keys {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		requests[i] = req
	}

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requests[b.LoopN()%len(requests)])
	}
}

// BenchmarkKeyedRateLimiter_Reject measures throughput on the reject path
// (limit exceeded). This is the path that writes 429 + Retry-After.
func BenchmarkKeyedRateLimiter_Reject(b *testing.B) {
	cfg := KeyedRateLimiterConfig{
		Limit:        1,
		Window:       time.Hour,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Warm up: first request is allowed; subsequent ones rejected.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	warmup := httptest.NewRecorder()
	handler.ServeHTTP(warmup, req)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkKeyedRateLimiter_HighCardinality measures throughput when keys
// rotate rapidly (high cardinality, low reuse). Exercises the heap eviction
// path under load.
func BenchmarkKeyedRateLimiter_HighCardinality(b *testing.B) {
	cfg := KeyedRateLimiterConfig{
		Limit:        100_000,
		Window:       time.Minute,
		MaxKeys:      10_000,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0." + strconv.Itoa(i%10_000) + ":1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkKeyedRateLimiter_EmptyKey measures the overhead of the empty-key
// exempt path (KeyExtractor returns ""). This should be near-zero — the
// middleware short-circuits before the limiter is consulted.
func BenchmarkKeyedRateLimiter_EmptyKey(b *testing.B) {
	cfg := KeyedRateLimiterConfig{
		Limit:  1,
		Window: time.Minute,
		KeyExtractor: func(_ *http.Request) string {
			return ""
		},
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkKeyedRateLimiter_EvictionOverhead measures the cost of TTL-based
// eviction. EvictionTTL is enabled and short, so each request triggers
// a sweep of the entries.
func BenchmarkKeyedRateLimiter_EvictionOverhead(b *testing.B) {
	cfg := KeyedRateLimiterConfig{
		Limit:        100_000,
		Window:       time.Minute,
		TTL:          10 * time.Millisecond, // aggressive TTL
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkKeyedRateLimiter_ClientIPExtractor measures the cost of using
// ClientIP as the key extractor (which respects X-Forwarded-For / X-Real-IP).
// This is the recommended production configuration behind a reverse proxy.
func BenchmarkKeyedRateLimiter_ClientIPExtractor(b *testing.B) {
	cfg := KeyedRateLimiterConfig{
		Limit:        100_000,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromClientIP(),
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 70.41.3.18, 150.172.238.178")
	req.RemoteAddr = "10.0.0.1:1234"

	b.ReportAllocs()

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}