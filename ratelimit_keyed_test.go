package httputil

import (
	"errors"
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

	handler := KeyedRateLimiterMiddleware(
		cfg,
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

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

	handler := KeyedRateLimiterMiddleware(
		cfg,
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

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

	handler := KeyedRateLimiterMiddleware(
		cfg,
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

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

// ---------------------------------------------------------------------------
// Coverage closure tests for KeyedRateLimiter
// ---------------------------------------------------------------------------

func TestKeyedRateLimiterMiddleware_OnAllowedCallback(t *testing.T) {
	t.Parallel()

	var allowedCount int

	cfg := KeyedRateLimiterConfig{
		Limit:        100,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
		OnAllowed: func(_ *http.Request) {
			allowedCount++
		},
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	handler.ServeHTTP(rec, req)

	if allowedCount != 1 {
		t.Fatalf("expected OnAllowed called once, got %d", allowedCount)
	}
}

func TestKeyedRateLimiterMiddleware_OnRejectedCallback(t *testing.T) {
	t.Parallel()

	var rejectedCount int

	var capturedRetryAfter string

	cfg := KeyedRateLimiterConfig{
		Limit:        1,
		Window:       time.Minute,
		Burst:        1,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
		OnRejected: func(_ *http.Request, retryAfter string) {
			rejectedCount++
			capturedRetryAfter = retryAfter
		},
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	// First request: allowed.
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Second request: rejected.
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if rejectedCount != 1 {
		t.Fatalf("expected OnRejected called once, got %d", rejectedCount)
	}

	if capturedRetryAfter == "" {
		t.Fatal("expected non-empty retryAfter")
	}
}

func TestKeyedRateLimiterMiddleware_RejectionHandler(t *testing.T) {
	t.Parallel()

	var handlerCalled bool

	cfg := KeyedRateLimiterConfig{
		Limit:        1,
		Window:       time.Minute,
		Burst:        1,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
		RejectionHandler: func(w http.ResponseWriter, _ *http.Request, _ string) {
			handlerCalled = true
			w.WriteHeader(http.StatusServiceUnavailable)
		},
	}

	handler := KeyedRateLimiterMiddleware(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	// First request: allowed.
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Second request: rejected by custom handler.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Fatal("custom RejectionHandler was not called")
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from custom handler, got %d", rec.Code)
	}
}

func TestPerKeyLimiter_EvictStale(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:        100,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
		TTL:          100 * time.Millisecond,
	}

	rl := NewKeyedRateLimiter(cfg)

	// Create entries for multiple keys.
	for _, addr := range []string{"1.1.1.1:1", "2.2.2.2:2"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		_, _ = rl.Check(req)
	}

	if count := rl.ActiveKeys(); count != 2 {
		t.Fatalf("expected 2 keys, got %d", count)
	}

	// Wait for TTL to expire.
	time.Sleep(200 * time.Millisecond)

	// Access a NEW key, which triggers eviction of stale entries.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "3.3.3.3:3"
	_, _ = rl.Check(req)

	// The stale entries should have been evicted; only the new key remains.
	if count := rl.ActiveKeys(); count > 1 {
		t.Fatalf("expected stale entries evicted, ActiveKeys=%d", count)
	}
}

func TestPerKeyLimiter_ReaccessAfterTTL(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:        100,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
		TTL:          100 * time.Millisecond,
	}

	rl := NewKeyedRateLimiter(cfg)

	// Create entry.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.1.1.1:1"
	_, _ = rl.Check(req)

	// Wait for TTL.
	time.Sleep(200 * time.Millisecond)

	// Re-access same key — entry is stale, should be refreshed.
	_, _ = rl.Check(req)

	if count := rl.ActiveKeys(); count != 1 {
		t.Fatalf("expected 1 active key after re-access, got %d", count)
	}
}

func TestKeyExtractorFromClientIP(t *testing.T) {
	t.Parallel()

	extractor := KeyExtractorFromClientIP()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	if got := extractor(req); got == "" {
		t.Fatal("expected non-empty key from ClientIP extractor")
	}
}

func TestKeyedRateLimiterConfigValidate_Default(t *testing.T) {
	t.Parallel()

	cfg := DefaultKeyedRateLimiterConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for default config", err)
	}
}

func TestKeyedRateLimiterConfigValidate_ZeroLimit(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:  0,
		Window: time.Minute,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for Limit=0")
	}

	if !errors.Is(err, errKeyedLimitZero) {
		t.Errorf("Validate() error = %v, want errKeyedLimitZero", err)
	}
}

func TestKeyedRateLimiterConfigValidate_NonPositiveWindow(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:  10,
		Window: 0,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for Window=0")
	}

	if !errors.Is(err, errKeyedWindowZero) {
		t.Errorf("Validate() error = %v, want errKeyedWindowZero", err)
	}
}

func TestKeyedRateLimiterConfigValidate_NegativeWindow(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:  10,
		Window: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative Window")
	}

	if !errors.Is(err, errKeyedWindowZero) {
		t.Errorf("Validate() error = %v, want errKeyedWindowZero", err)
	}
}

func TestKeyedRateLimiterConfigValidate_AllowsBurstZero(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:  10,
		Window: time.Minute,
		Burst:  0, // defaults to Limit at construction — validation must allow
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (Burst=0 is allowed)", err)
	}
}

func TestKeyedRateLimiterConfigValidate_NegativeTTL(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:  10,
		Window: time.Minute,
		TTL:    -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative TTL")
	}

	if !errors.Is(err, errKeyedTTLNegative) {
		t.Errorf("Validate() error = %v, want errKeyedTTLNegative", err)
	}
}

func TestKeyedRateLimiterConfigValidate_AllowsZeroTTL(t *testing.T) {
	t.Parallel()

	cfg := KeyedRateLimiterConfig{
		Limit:  10,
		Window: time.Minute,
		TTL:    0, // defaults to 10 min at construction — validation must allow
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (TTL=0 is allowed)", err)
	}
}
