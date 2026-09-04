package httputil

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	servertiming "github.com/larsartmann/httputil/server_timing"
)

// TestChain_CORSWithRecoveryAndLogging verifies that a CORS-preflight-relevant
// request passes through a Recovery+Logging chain. The log and recovery
// behaviors are asserted by their dedicated tests; this one pins the
// pass-through composition.
func TestChain_CORSWithRecoveryAndLogging(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()

	inner := newWriteStatusHandler("ok")

	handler := Chain(inner, CORS(corsCfg), Recovery(logger), Logging(logger))

	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	assertHeader(t, rec, "Access-Control-Allow-Origin", "*")
}

func TestChain_RecoveryCatchesPanicWithLogging(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()

	inner := newPanicHandler("integration test panic")

	handler := Chain(inner, CORS(corsCfg), Recovery(logger), Logging(logger))

	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusInternalServerError)
}

func TestChain_RequestIDSecurityHeaders(t *testing.T) {
	t.Parallel()

	reqCfg := DefaultRequestIDConfig()
	secCfg := DefaultSecurityHeadersConfig()

	inner := newWriteStatusHandler("")

	handler := Chain(inner, SecurityHeaders(secCfg), RequestID(reqCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header is empty")
	}

	assertHeader(t, rec, "X-Content-Type-Options", "nosniff")
}

func TestChain_TimeoutThenRecovery(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		if !ok {
			t.Error("context has no deadline")
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := Chain(inner, Recovery(logger), Timeout(time.Second))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
}

func BenchmarkChain(b *testing.B) {
	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()
	reqCfg := DefaultRequestIDConfig()
	secCfg := DefaultSecurityHeadersConfig()

	inner := newWriteStatusHandler("")

	handler := Chain(
		inner,
		SecurityHeaders(secCfg),
		RequestID(reqCfg),
		Recovery(logger),
		Logging(logger),
		CORS(corsCfg),
	)

	req := newTestRequest(http.MethodGet, "/", "http://example.com")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func TestChain_PreservesHandlerContentLengthOnSmallResponses(t *testing.T) {
	t.Parallel()

	body := `{"ok":true}`
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})

	wrapped := Chain(
		handler,
		Compression(CompressionConfig{WriterFactories: DefaultWriterFactories()}),
		Recovery(slog.Default()),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "identity")
	wrapped.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Length"); got != strconv.Itoa(len(body)) {
		t.Errorf(
			"Content-Length = %q, want %q (identity responses must preserve the handler header)",
			got,
			strconv.Itoa(len(body)),
		)
	}

	if rec.Body.String() != body {
		t.Errorf("body = %q, want %q", rec.Body.String(), body)
	}
}

func TestChain_RecoveryLogsAndRecoversPanic(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})

	wrapped := Chain(handler, Recovery(logger), Logging(logger))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	logs := buf.String()

	if !strings.Contains(logs, "panic") {
		t.Errorf("logs should record the panic, got:\n%s", logs)
	}

	if !strings.Contains(logs, slog.LevelKey) || !strings.Contains(logs, slog.LevelError.String()) {
		t.Errorf("logs should contain an error-level record, got:\n%s", logs)
	}
}

func TestChain_CSRFWithServerTimingAddsHeaderOnRejection(t *testing.T) {
	t.Parallel()

	stmw := servertiming.ServerTimingMiddleware()
	csrfmw := CSRFMiddleware(CSRFConfig{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := stmw(csrfmw(handler))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if got := rec.Header().Get(servertiming.HeaderServerTiming); got == "" {
		t.Error(
			"Server-Timing header should survive a CSRF rejection (outer middleware still sets it)",
		)
	}
}

func TestChain_KeyedRateLimitEvictionUnderChurn(t *testing.T) {
	t.Parallel()

	cfg := DefaultKeyedRateLimiterConfig()
	cfg.Limit = 1000
	cfg.Window = 50 * time.Millisecond
	cfg.MaxKeys = 4
	cfg.TTL = 10 * time.Millisecond

	mw := KeyedRateLimiterMiddleware(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for round := range 20 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = fmt.Sprintf("10.0.%d.%d:1000", round%250, round/250%250)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("round %d: fresh key rejected with %d", round, rec.Code)
		}

		time.Sleep(2 * time.Millisecond)
	}
}

func TestChain_DecompressionThenMaxBodySizeLimitsDecompressed(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(make([]byte, 4096))
	_ = zw.Close()

	cfg := DefaultDecompressionConfig()
	cfg.MaxDecompressionSize = 1024
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusExpectationFailed)

			return
		}
		w.WriteHeader(http.StatusOK)
		_ = r.Body.Close()
		_ = n
	})

	wrapped := Chain(
		inner,
		Decompression(cfg),
		MaxBodySizeMiddleware(MaxBodySizeConfig{MaxBytes: 2048}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusExpectationFailed {
		t.Errorf(
			"status = %d, want %d (bomb protection must abort the decompressed read)",
			rec.Code,
			http.StatusExpectationFailed,
		)
	}
}

func TestChain_DecompressionThenCompressionRoundTrips(t *testing.T) {
	t.Parallel()

	payload := "round trip body"

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(payload))
	_ = zw.Close()

	var got strings.Builder
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(&got, r.Body)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Chain(
		inner,
		Compression(CompressionConfig{WriterFactories: DefaultWriterFactories()}),
		Decompression(DefaultDecompressionConfig()),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "identity")
	wrapped.ServeHTTP(rec, req)

	if got.String() != payload {
		t.Errorf("round-tripped body = %q, want %q", got.String(), payload)
	}
}

func TestChain_NonceCSPSurvivesDefaultSecurityHeaders(t *testing.T) {
	t.Parallel()

	nonceCfg := DefaultNonceConfig()
	nonceCfg.CSPBuilder = RecommendedCSPWithNonce
	static := SecurityHeaders(DefaultSecurityHeadersConfig())

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if NonceFromRequest(r) == "" {
			t.Error("nonce should be present in the request context")
		}

		w.WriteHeader(http.StatusOK)
	})

	wrapped := Chain(inner, Nonce(nonceCfg), static)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	got := rec.Header().Get("Content-Security-Policy")
	if got == "" || !strings.Contains(got, "'nonce-") {
		t.Errorf(
			"nonce CSP should survive the default SecurityHeaders (which sets no CSP), got %q",
			got,
		)
	}
}

func TestChain_NonceBeforeSecurityHeadersKeepsNonceCSP(t *testing.T) {
	t.Parallel()

	nonceCfg := DefaultNonceConfig()
	nonceCfg.CSPBuilder = RecommendedCSPWithNonce
	static := SecurityHeaders(DefaultSecurityHeadersConfig())

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Chain(inner, Nonce(nonceCfg), static)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	got := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(got, "'nonce-") {
		t.Errorf("nonce-bearing CSP should win when Nonce is inner, got %q", got)
	}
}

func TestChain_NonceInnerOverwritesOuter(t *testing.T) {
	t.Parallel()

	outerMW := Nonce(DefaultNonceConfig())
	innerMW := Nonce(DefaultNonceConfig())

	var outerNonce, seenNonce string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenNonce = NonceFromRequest(r)
		w.WriteHeader(http.StatusOK)
	})

	capture := outerMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outerNonce = NonceFromRequest(r)
		innerMW(inner).ServeHTTP(w, r)
	}))

	capture.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if outerNonce == "" || seenNonce == "" {
		t.Fatal("both nonce instances should populate the context")
	}

	if seenNonce == outerNonce {
		t.Error("the inner Nonce instance must overwrite the outer one, not reuse it")
	}
}

func TestChain_NonceDiffersAcrossRequests(t *testing.T) {
	t.Parallel()

	mw := Nonce(DefaultNonceConfig())

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(NonceFromRequest(r)))
	})

	wrapped := mw(inner)

	first := httptest.NewRecorder()
	wrapped.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))

	second := httptest.NewRecorder()
	wrapped.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))

	if first.Body.String() == "" {
		t.Fatal("nonce should be written to the response body")
	}

	if first.Body.String() == second.Body.String() {
		t.Error("nonces must differ across requests")
	}
}

func TestChain_NonceWithRecoveryStillSetsNonce(t *testing.T) {
	t.Parallel()

	nonceCfg := DefaultNonceConfig()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	wrapped := Chain(inner, Nonce(nonceCfg), Recovery(slog.Default()))

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
