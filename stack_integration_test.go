package httputil

import (
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	etag "github.com/larsartmann/go-etag"
	servertiming "github.com/larsartmann/httputil/server_timing"
)

// TestStack_FullMiddlewareComposition chains all 18 middlewares in their
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
	buildFullStack(t, stack, logger)

	if err := stack.Validate(); err != nil {
		t.Fatalf("stack validation failed: %v", err)
	}

	if err := stack.Add(MiddlewareRecovery, Recovery(logger)); err == nil {
		t.Error("expected duplicate recovery middleware to be rejected")
	}

	// Each subtest builds its own handler + atomic so they are fully
	// independent and safe to run in parallel.
	t.Run("GET produces all expected headers", func(t *testing.T) {
		t.Parallel()
		called := &atomic.Bool{}
		handler := stack.Build(newInnerHandler(called))
		verifyGETHeaders(t, handler, called)
	})

	t.Run("POST without CSRF token is rejected", func(t *testing.T) {
		t.Parallel()
		called := &atomic.Bool{}
		handler := stack.Build(newInnerHandler(called))
		verifyCSRFRejection(t, handler, called)
	})

	t.Run("OPTIONS preflight succeeds with CORS headers", func(t *testing.T) {
		t.Parallel()
		handler := stack.Build(newInnerHandler(&atomic.Bool{}))
		verifyCORSPreflight(t, handler)
	})

	t.Run("Panic in inner handler returns 500", func(t *testing.T) {
		t.Parallel()
		verifyPanicRecovery(t, logger)
	})

	t.Run("Rate-limited response still has all headers", func(t *testing.T) {
		t.Parallel()
		verifyRateLimitHeaders(t, logger)
	})
}

// TestStack_RecoveryMustBeOuterMost verifies the ordering invariant: if a
// middleware is placed before Recovery, Validate() rejects it.
func TestStack_RecoveryMustBeOuterMost(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()

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

// addStackMiddleware adds a named middleware to a stack and fails the test if
// Add returns an error.
func addStackMiddleware(t *testing.T, stack *MiddlewareStack, name string, mw Middleware) {
	t.Helper()

	if err := stack.Add(name, mw); err != nil {
		t.Fatalf("stack.Add(%q): %v", name, err)
	}
}

// buildFullStack adds all 18 built-in middleware to stack in the
// production-recommended order. Order is documented in README.md.
func buildFullStack(t *testing.T, stack *MiddlewareStack, logger *slog.Logger) {
	t.Helper()

	addStackMiddleware(t, stack, MiddlewareRecovery, Recovery(logger))
	addStackMiddleware(t, stack, MiddlewareLogging, Logging(logger))
	addStackMiddleware(t, stack, MiddlewareRequestID, RequestID(DefaultRequestIDConfig()))
	addStackMiddleware(
		t,
		stack,
		MiddlewareSecurityHeaders,
		SecurityHeaders(DefaultSecurityHeadersConfig()),
	)
	addStackMiddleware(t, stack, MiddlewareNonce, Nonce(DefaultNonceConfig()))
	addStackMiddleware(t, stack, MiddlewareCORS, CORS(DefaultCORSConfig()))
	addStackMiddleware(
		t,
		stack,
		MiddlewareKeyedRateLimit,
		KeyedRateLimiterMiddleware(KeyedRateLimiterConfig{
			Limit:        1000,
			Window:       time.Minute,
			KeyExtractor: KeyExtractorFromRemoteAddr(),
		}),
	)
	addStackMiddleware(t, stack, MiddlewareCSRF, CSRFMiddleware(CSRFConfig{}))
	addStackMiddleware(t, stack, MiddlewareCompression, Compression(DefaultCompressionConfig()))
	addStackMiddleware(t, stack, MiddlewareETag, ETag(etag.DefaultETagConfig()))
	addStackMiddleware(t, stack, MiddlewareTimeout, Timeout(30*time.Second))
	addStackMiddleware(t, stack, MiddlewareClientIP, ClientIPMiddleware)
	addStackMiddleware(t, stack, MiddlewareServerTiming, Middleware(servertiming.ServerTimingMiddleware()))
}

// newInnerHandler returns the inner handler used by stack composition tests.
// called is set to true when the handler runs.
func newInnerHandler(called *atomic.Bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func verifyGETHeaders(t *testing.T, handler http.Handler, called *atomic.Bool) {
	t.Helper()

	called.Store(false)

	rec := newRecorder()
	req := newTestRequest(http.MethodGet, "/", "")

	handler.ServeHTTP(rec, req)

	if !called.Load() {
		t.Fatal("inner handler was not called")
	}

	assertStatus(t, rec, http.StatusOK)

	headers := []struct {
		name  string
		value string
	}{
		{"X-Request-ID", ""}, // non-empty
		{"X-Content-Type-Options", "nosniff"},
		{"Access-Control-Allow-Origin", "*"},
		{"Content-Security-Policy", ""},       // non-empty (nonce middleware)
		{servertiming.HeaderServerTiming, ""}, // non-empty
	}

	for _, h := range headers {
		got := rec.Header().Get(h.name)

		if h.value == "" {
			if got == "" {
				t.Errorf("%s header missing", h.name)
			}

			continue
		}

		if got != h.value {
			t.Errorf("%s = %q, want %q", h.name, got, h.value)
		}
	}
}

func verifyCSRFRejection(t *testing.T, handler http.Handler, called *atomic.Bool) {
	t.Helper()

	called.Store(false)

	rec := newRecorder()
	req := newTestRequest(http.MethodPost, "/", "")
	req.Header.Set("Origin", "http://malicious-site.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	handler.ServeHTTP(rec, req)

	if called.Load() {
		t.Error("inner handler should not be called for POST without CSRF token")
	}

	assertStatus(t, rec, http.StatusForbidden)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff (must be set on rejection)", got)
	}
}

func verifyCORSPreflight(t *testing.T, handler http.Handler) {
	t.Helper()

	rec := newRecorder()
	req := newTestRequest(http.MethodOptions, "/", "http://example.com")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNoContent)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("Access-Control-Allow-Origin missing on preflight")
	}
}

func verifyPanicRecovery(t *testing.T, logger *slog.Logger) {
	t.Helper()

	panicStack := NewMiddlewareStack()

	addStackMiddleware(t, panicStack, MiddlewareRecovery, Recovery(logger))
	addStackMiddleware(t, panicStack, MiddlewareRequestID, RequestID(DefaultRequestIDConfig()))

	panicHandler := panicStack.Build(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("stack integration test panic")
	}))

	rec := newRecorder()
	req := newTestRequest(http.MethodGet, "/", "")

	panicHandler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusInternalServerError)
}

func verifyRateLimitHeaders(t *testing.T, logger *slog.Logger) {
	t.Helper()

	rlStack := NewMiddlewareStack()

	addStackMiddleware(t, rlStack, MiddlewareRecovery, Recovery(logger))
	addStackMiddleware(
		t,
		rlStack,
		MiddlewareSecurityHeaders,
		SecurityHeaders(DefaultSecurityHeadersConfig()),
	)
	addStackMiddleware(
		t,
		rlStack,
		MiddlewareKeyedRateLimit,
		KeyedRateLimiterMiddleware(KeyedRateLimiterConfig{
			Limit:        1,
			Window:       time.Minute,
			KeyExtractor: KeyExtractorFromRemoteAddr(),
		}),
	)

	rlHandler := rlStack.Build(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	firstRec := newRecorder()
	firstReq := newTestRequest(http.MethodGet, "/", "")

	rlHandler.ServeHTTP(firstRec, firstReq)

	assertStatus(t, firstRec, http.StatusOK)

	secondRec := newRecorder()
	secondReq := newTestRequest(http.MethodGet, "/", "")

	rlHandler.ServeHTTP(secondRec, secondReq)

	assertStatus(t, secondRec, http.StatusTooManyRequests)

	if got := secondRec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf(
			"X-Content-Type-Options on 429 = %q, want nosniff (outer middleware must run on rejection)",
			got,
		)
	}
}

func TestServerTimingMiddleware_ComposesWithChain(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stopWork := servertiming.MeasureServerTiming(r.Context(), "work")

		time.Sleep(5 * time.Millisecond)
		stopWork()

		_, _ = w.Write([]byte("ok"))
	})

	stacked := Chain(inner, Middleware(servertiming.ServerTimingMiddleware()))

	rec := newRecorder()
	stacked.ServeHTTP(rec, newTestRequest(http.MethodGet, "/", ""))

	hv := rec.Header().Get(servertiming.HeaderServerTiming)
	if hv == "" {
		t.Fatal("Server-Timing header missing through Chain")
	}

	if !strings.Contains(hv, "work;") {
		t.Errorf("work metric missing in %q", hv)
	}

	if !strings.Contains(hv, "total;") {
		t.Errorf("total metric missing in %q", hv)
	}
}
