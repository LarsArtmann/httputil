package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenBucketLimiterAllowsWithinBurst(t *testing.T) {
	t.Parallel()

	limiter, err := NewTokenBucketLimiter(1, 2)
	if err != nil {
		t.Fatal(err)
	}

	if !limiter.Allow("key1") {
		t.Error("first Allow should succeed within burst")
	}

	if !limiter.Allow("key1") {
		t.Error("second Allow should succeed within burst")
	}

	if limiter.Allow("key1") {
		t.Error("third Allow should fail after burst exhausted")
	}
}

func TestTokenBucketLimiterKeysAreIndependent(t *testing.T) {
	t.Parallel()

	limiter, err := NewTokenBucketLimiter(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !limiter.Allow("key1") {
		t.Error("Allow key1 should succeed")
	}

	if !limiter.Allow("key2") {
		t.Error("Allow key2 should succeed with independent bucket")
	}
}

func TestRateLimitAllowsWithinLimit(t *testing.T) {
	t.Parallel()

	limiter, err := NewTokenBucketLimiter(100, 10)
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultRateLimitConfig()
	cfg.Limiter = limiter

	handler := RateLimit(cfg)(newStatusOnlyHandler(http.StatusOK))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRateLimitDeniesWhenExceeded(t *testing.T) {
	t.Parallel()

	limiter, err := NewTokenBucketLimiter(0.01, 1)
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultRateLimitConfig()
	cfg.Limiter = limiter

	handler := RateLimit(cfg)(newStatusOnlyHandler(http.StatusOK))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.2.3.4:1234"

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("first request status = %d, want %d", rec1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "1.2.3.4:1234"

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request status = %d, want %d", rec2.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitCustomKeyFunc(t *testing.T) {
	t.Parallel()

	limiter, err := NewTokenBucketLimiter(0.01, 1)
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultRateLimitConfig()
	cfg.Limiter = limiter
	cfg.KeyFunc = func(r *http.Request) string {
		return r.Header.Get("X-Api-Key")
	}

	handler := RateLimit(cfg)(newStatusOnlyHandler(http.StatusOK))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Api-Key", "key-a")
	req1.RemoteAddr = "1.2.3.4:1234"

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first key status = %d, want %d", rec1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Api-Key", "key-b")
	req2.RemoteAddr = "1.2.3.4:1234"

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf(
			"different key should have own bucket: status = %d, want %d",
			rec2.Code,
			http.StatusOK,
		)
	}
}

func TestRateLimitCustomOnDenied(t *testing.T) {
	t.Parallel()

	limiter, err := NewTokenBucketLimiter(0.01, 1)
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultRateLimitConfig()
	cfg.Limiter = limiter
	cfg.OnDenied = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"slow down"}`))
	})

	handler := RateLimit(cfg)(newStatusOnlyHandler(http.StatusOK))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.2.3.4:1234"

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "1.2.3.4:1234"

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", rec2.Code, http.StatusTooManyRequests)
	}

	if rec2.Body.String() != `{"error":"slow down"}` {
		t.Errorf("body = %q, want JSON error response", rec2.Body.String())
	}
}

func TestRateLimitConfigValidateRejectsNilLimiter(t *testing.T) {
	t.Parallel()

	cfg := DefaultRateLimitConfig()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nil Limiter, got nil")
	}
}

func TestNewTokenBucketLimiterRejectsNonPositiveRate(t *testing.T) {
	t.Parallel()

	_, err := NewTokenBucketLimiter(0, 5)
	if err == nil {
		t.Fatal("expected error for zero rate, got nil")
	}

	_, err = NewTokenBucketLimiter(-1, 5)
	if err == nil {
		t.Fatal("expected error for negative rate, got nil")
	}
}

func TestNewTokenBucketLimiterRejectsNonPositiveBurst(t *testing.T) {
	t.Parallel()

	_, err := NewTokenBucketLimiter(5, 0)
	if err == nil {
		t.Fatal("expected error for zero burst, got nil")
	}

	_, err = NewTokenBucketLimiter(5, -1)
	if err == nil {
		t.Fatal("expected error for negative burst, got nil")
	}
}

func TestTokenBucketLimiterEvictsIdleBuckets(t *testing.T) {
	t.Parallel()

	clock := time.Now()

	limiter, err := NewTokenBucketLimiter(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	limiter.EvictionTTL = 5 * time.Minute
	limiter.now = func() time.Time { return clock }

	limiter.Allow("active")
	limiter.Allow("idle")

	if len(limiter.buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(limiter.buckets))
	}

	clock = clock.Add(6 * time.Minute)

	limiter.Allow("active")

	if _, ok := limiter.buckets["idle"]; ok {
		t.Error("expected idle bucket to be evicted after TTL")
	}

	if _, ok := limiter.buckets["active"]; !ok {
		t.Error("expected active bucket to still exist after sweep")
	}
}

func TestTokenBucketLimiterNoEvictionByDefault(t *testing.T) {
	t.Parallel()

	clock := time.Now()

	limiter, err := NewTokenBucketLimiter(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	limiter.now = func() time.Time { return clock }

	limiter.Allow("key1")

	clock = clock.Add(24 * time.Hour)

	limiter.Allow("key2")

	if len(limiter.buckets) != 2 {
		t.Errorf("expected 2 buckets (no eviction by default), got %d", len(limiter.buckets))
	}
}
