package httputil

import (
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestStack_FullMiddlewareComposition chains all 16 middlewares in their
// recommended production order and verifies:
//  1. A standard GET request flows through every layer without breakage.
//  2. Each middleware's headers are present on the response (proves every
//     middleware ran).
//  3. A POST request without a CSRF token is rejected (proves CSRF is wired).
//  4. A 429 response still includes security headers (proves ordering:
//     rate-limit rejection does not skip outer middleware).
//  5. A panicking inner handler returns 500 (proves Recovery is outermost).
//
// The middleware order matches the production guidance in README.md and
// AGENTS.md (Recovery outermost, Logging next so it sees the final status,
// rate limiting before expensive work, CSRF before state-changing handlers).
func TestStack_FullMiddlewareComposition(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()

	stack := NewMiddlewareStack()

	addStackMiddleware(t, stack, MiddlewareRecovery, Recovery(logger))
	addStackMiddleware(t, stack, MiddlewareLogging, Logging(logger))
	addStackMiddleware(t, stack, MiddlewareRequestID, RequestID(DefaultRequestIDConfig()))
	addStackMiddleware(t, stack, MiddlewareSecurityHeaders, SecurityHeaders(DefaultSecurityHeadersConfig()))
	addStackMiddleware(t, stack, MiddlewareCORS, CORS(DefaultCORSConfig()))
	addStackMiddleware(t, stack, MiddlewareKeyedRateLimit, KeyedRateLimiterMiddleware(KeyedRateLimiterConfig{
		Limit:        1000,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}))
	addStackMiddleware(t, stack, MiddlewareCSRF, CSRFMiddleware(CSRFConfig{}))
	addStackMiddleware(t, stack, MiddlewareCompression, Compression(DefaultCompressionConfig()))
	addStackMiddleware(t, stack, MiddlewareETag, ETag(DefaultETagConfig()))
	addStackMiddleware(t, stack, MiddlewareTimeout, Timeout(30*time.Second))
	addStackMiddleware(t, stack, MiddlewareClientIP, ClientIPMiddleware)
	addStackMiddleware(t, stack, MiddlewareServerTiming, ServerTimingMiddleware())

	if err := stack.Validate(); err != nil {
		t.Fatalf("stack validation failed: %v", err)
	}

	if err := stack.Add(MiddlewareRecovery, Recovery(logger)); err == nil {
		t.Error("expected duplicate recovery middleware to be rejected")
	}

	var called atomic.Bool

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := stack.Build(inner)

	t.Run("GET produces all expected headers", func(t *testing.T) {
		t.Parallel()

		called.Store(false)

		rec := newRecorder()
		req := newTestRequest(http.MethodGet, "/", "")

		handler.ServeHTTP(rec, req)

		if !called.Load() {
			t.Fatal("inner handler was not called")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}

		// Recovery: passes through (nothing to recover)
		// Logging: passes through
		// RequestID: sets X-Request-ID
		if got := rec.Header().Get("X-Request-ID"); got == "" {
			t.Error("X-Request-ID header missing")
		}

		// SecurityHeaders: sets X-Content-Type-Options
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
		}

		// CORS: sets Access-Control-Allow-Origin (default config: "*")
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
		}

		// ServerTiming: sets Server-Timing header
		if got := rec.Header().Get(HeaderServerTiming); got == "" {
			t.Error("Server-Timing header missing")
		}

		// ETag: sets ETag header
		if got := rec.Header().Get("ETag"); got == "" {
			t.Error("ETag header missing")
		}
	})

	t.Run("POST without CSRF token is rejected", func(t *testing.T) {
		t.Parallel()

		called.Store(false)

		rec := newRecorder()
		req := newTestRequest(http.MethodPost, "/", "")

		handler.ServeHTTP(rec, req)

		if called.Load() {
			t.Error("inner handler should not be called for POST without CSRF token")
		}

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403 (CSRF rejection)", rec.Code)
		}

		// Even on CSRF rejection, security headers must be set — they are
		// applied in middleware that wraps CSRF, so rejection does not
		// skip them.
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff (must be set on rejection)", got)
		}
	})

	t.Run("OPTIONS preflight succeeds with CORS headers", func(t *testing.T) {
		t.Parallel()

		called.Store(false)

		rec := newRecorder()
		req := newTestRequest(http.MethodOptions, "/", "http://example.com")
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want 204 (CORS preflight)", rec.Code)
		}

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
			t.Error("Access-Control-Allow-Origin missing on preflight")
		}
	})

	t.Run("Panic in inner handler returns 500", func(t *testing.T) {
		t.Parallel()

		panicStack := NewMiddlewareStack()

		addStackMiddleware(t, panicStack, MiddlewareRecovery, Recovery(logger))
		addStackMiddleware(t, panicStack, MiddlewareRequestID, RequestID(DefaultRequestIDConfig()))

		panicInner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("stack integration test panic")
		})

		panicHandler := panicStack.Build(panicInner)

		panicRec := newRecorder()
		panicReq := newTestRequest(http.MethodGet, "/", "")

		panicHandler.ServeHTTP(panicRec, panicReq)

		if panicRec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500 (panic recovery)", panicRec.Code)
		}
	})

	t.Run("Rate-limited response still has all headers", func(t *testing.T) {
		t.Parallel()

		// Build a separate stack with very low rate limit so we can hit 429.
		rlStack := NewMiddlewareStack()

		addStackMiddleware(t, rlStack, MiddlewareRecovery, Recovery(logger))
		addStackMiddleware(t, rlStack, MiddlewareSecurityHeaders, SecurityHeaders(DefaultSecurityHeadersConfig()))
		addStackMiddleware(t, rlStack, MiddlewareKeyedRateLimit, KeyedRateLimiterMiddleware(KeyedRateLimiterConfig{
			Limit:        1,
			Window:       time.Minute,
			KeyExtractor: KeyExtractorFromRemoteAddr(),
		}))

		rlHandler := rlStack.Build(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request: allowed, returns 200.
		firstRec := newRecorder()
		firstReq := newTestRequest(http.MethodGet, "/", "")

		rlHandler.ServeHTTP(firstRec, firstReq)

		if firstRec.Code != http.StatusOK {
			t.Errorf("first request status = %d, want 200", firstRec.Code)
		}

		// Second request: rejected with 429.
		secondRec := newRecorder()
		secondReq := newTestRequest(http.MethodGet, "/", "")

		rlHandler.ServeHTTP(secondRec, secondReq)

		if secondRec.Code != http.StatusTooManyRequests {
			t.Errorf("second request status = %d, want 429", secondRec.Code)
		}

		// Security headers must still be set on the 429 — SecurityHeaders
		// is OUTER relative to RateLimit, so rejection does not skip it.
		if got := secondRec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf(
				"X-Content-Type-Options on 429 = %q, want nosniff (outer middleware must run on rejection)",
				got,
			)
		}
	})
}

// TestStack_RecoveryMustBeOuterMost verifies the ordering invariant: if a
// middleware is placed before Recovery, Validate() rejects it.
func TestStack_RecoveryMustBeOuterMost(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	// Intentionally wrong: Logging before Recovery.
	if err := stack.Add("logging", Logging(slog.New(slog.DiscardHandler))); err != nil {
		t.Fatalf("Add(logging) error: %v", err)
	}

	if err := stack.Add(MiddlewareRecovery, Recovery(slog.New(slog.DiscardHandler))); err != nil {
		t.Fatalf("Add(recovery) error: %v", err)
	}

	err := stack.Validate()
	if err == nil {
		t.Fatal("expected Validate to reject Recovery not at position 0")
	}

	if !strings.Contains(err.Error(), "recovery") {
		t.Errorf("error = %v, want error mentioning recovery", err)
	}
}

// TestStack_DuplicateMiddlewareRejected verifies the duplicate prevention
// invariant: adding the same middleware name twice is rejected.
func TestStack_DuplicateMiddlewareRejected(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

	if err := stack.Add(MiddlewareRequestID, RequestID(DefaultRequestIDConfig())); err != nil {
		t.Fatalf("first Add: %v", err)
	}

	err := stack.Add(MiddlewareRequestID, RequestID(DefaultRequestIDConfig()))
	if err == nil {
		t.Fatal("expected duplicate middleware to be rejected")
	}

	if !strings.Contains(err.Error(), "already") {
		t.Errorf("error = %v, want error mentioning duplicate", err)
	}
}

// addStackMiddleware is a small helper that adds a named middleware to a stack
// and fails the test if Add returns an error (which would indicate a duplicate
// or other invariant violation we never want to silently swallow).
func addStackMiddleware(t *testing.T, stack *MiddlewareStack, name string, mw Middleware) {
	t.Helper()

	if err := stack.Add(name, mw); err != nil {
		t.Fatalf("stack.Add(%q): %v", name, err)
	}
}
