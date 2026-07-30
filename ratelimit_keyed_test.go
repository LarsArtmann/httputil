package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestKeyedRateLimiterMiddleware_AllowsUnderLimit(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:        10,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}

	handler := KeyedRateLimiterMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for range 5 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	}
}

func TestKeyedRateLimiterMiddleware_RejectsOverLimit(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:        2,
		Window:       time.Minute,
		Burst:        2,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}

	handler := KeyedRateLimiterMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	for range 2 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header missing")
	}
}

func TestKeyedRateLimiter_EmptyKeyExempt(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:  1,
		Window: time.Minute,
		KeyExtractor: func(_ *http.Request) string {
			return ""
		},
	}

	handler := KeyedRateLimiterMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for range 10 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("empty key should always be allowed, got %d", rec.Code)
		}
	}
}

func TestNewKeyedRateLimiter_ActiveKeys(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:        100,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}

	rl := NewKeyedRateLimiter(cfg)

	for _, addr := range []string{"1.1.1.1:1", "2.2.2.2:2", "3.3.3.3:3"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		_, _ = rl.Check(req)
	}

	if count := rl.ActiveKeys(); count != 3 {
		t.Fatalf("ActiveKeys = %d, want 3", count)
	}
}

func TestKeyedRateLimiter_MaxKeysCap(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:        100,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
		MaxKeys:      2,
	}

	rl := NewKeyedRateLimiter(cfg)

	for _, addr := range []string{"1.1.1.1:1", "2.2.2.2:2", "3.3.3.3:3"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		_, _ = rl.Check(req)
	}

	if count := rl.ActiveKeys(); count > 2 {
		t.Fatalf("ActiveKeys = %d, want <= 2 (MaxKeys cap)", count)
	}
}

func TestEvictionHeapPushNonPtr(t *testing.T) {
	t.Parallel()

	h := &evictionHeap{}
	h.Push("not a pointer")

	if h.Len() != 0 {
		t.Errorf("Push(non-pointer) should be ignored, got len=%d", h.Len())
	}
}

func TestDefaultKeyedRateLimiterConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultKeyedRateLimiterConfig()
	if cfg.Limit != DefaultRateLimit {
		t.Errorf("Limit = %d, want %d", cfg.Limit, DefaultRateLimit)
	}

	if cfg.Window != DefaultRateWindow {
		t.Errorf("Window = %v, want %v", cfg.Window, DefaultRateWindow)
	}
}
