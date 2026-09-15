package httputil

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
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

	var readErr error
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		readErr = err
		if err != nil {
			w.WriteHeader(http.StatusExpectationFailed)

			return
		}

		w.WriteHeader(http.StatusOK)
		_ = r.Body.Close()
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

	if readErr == nil {
		t.Fatal("decompressed body read error = nil, want the bomb-protection size error")
	}

	if !errors.Is(readErr, errDecompressionSizeExceeded) {
		t.Errorf("read error = %v, want the decompression.size_exceeded sentinel", readErr)
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

// TestChain_NonceInnerToSecurityHeaders_OverwritesStaticCSP pins the
// documented composition guidance: with SecurityHeaders outermost carrying a
// static Content-Security-Policy and Nonce inner to it, the nonce-bearing CSP
// overwrites the static policy (both middlewares set the header on the request
// path, so the inner one writes last).
func TestChain_NonceInnerToSecurityHeaders_OverwritesStaticCSP(t *testing.T) {
	t.Parallel()

	const staticCSP = "default-src 'self'; report-uri /csp-reports"

	nonceCfg := DefaultNonceConfig()
	nonceCfg.CSPBuilder = RecommendedCSPWithNonce

	staticCfg := DefaultSecurityHeadersConfig()
	staticCfg.ContentSecurityPolicy = staticCSP

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Chain(inner, SecurityHeaders(staticCfg), Nonce(nonceCfg))

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	got := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(got, "'nonce-") {
		t.Errorf("nonce-bearing CSP should overwrite the static policy when Nonce is inner, got %q", got)
	}

	if strings.Contains(got, "report-uri") {
		t.Errorf("static CSP should not survive the nonce overwrite, got %q", got)
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

// TestChain_RecoveryErrAbortHandler_ThroughStack pins that the net/http
// ErrAbortHandler sentinel survives the full Chain, not just a bare Recovery
// wrapper: the sentinel must be re-panicked through the middleware stack so
// the server's silent connection-abort handling applies.
func TestChain_RecoveryErrAbortHandler_ThroughStack(t *testing.T) {
	t.Parallel()

	aborting := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(http.ErrAbortHandler)
	})

	handler := Chain(
		aborting,
		Nonce(DefaultNonceConfig()),
		Recovery(newTestLogger()),
	)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("Chain swallowed http.ErrAbortHandler, want it re-panicked")
		}

		err, ok := recovered.(error)
		if !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", recovered)
		}
	}()

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))
}

// cspNonceFromHeader extracts the first nonce value from a
// Content-Security-Policy header ('nonce-<value>' token) for cross-checking
// against nonces rendered into response bodies.
func cspNonceFromHeader(t *testing.T, csp string) string {
	t.Helper()

	marker := "'nonce-"

	start := strings.Index(csp, marker)
	if start < 0 {
		t.Fatalf("no 'nonce- token in CSP header %q", csp)
	}

	start += len(marker)

	end := strings.IndexByte(csp[start:], '\'')
	if end < 0 {
		t.Fatalf("unterminated 'nonce- token in CSP header %q", csp)
	}

	return csp[start : start+end]
}

// TestChain_NonceThenCompression_CSPNonceEmbeddedInCompressedBody runs the
// never-previously-exercised nonce x compression composition: the CSP header
// written through the compression wrapper must survive, and the nonce the
// handler rendered into the body must be exactly the nonce the CSP header
// carries, after a full gzip round-trip.
func TestChain_NonceThenCompression_CSPNonceEmbeddedInCompressedBody(t *testing.T) {
	t.Parallel()

	padding := strings.Repeat("x", defaultCompressionMinSize)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerContentType, "text/html")

		_, _ = fmt.Fprintf(w, "<script %s>ok()</script><!-- %s -->", NonceAttr(r), padding)
	})

	wrapped := Chain(inner, Compression(DefaultCompressionConfig()), Nonce(DefaultNonceConfig()))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()
	wrapped.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header = empty, want the nonce-bearing CSP")
	}

	cspNonce := cspNonceFromHeader(t, csp)

	if got := rec.Header().Get(headerContentEncoding); got != encodingGzip {
		t.Fatalf("Content-Encoding = %q, want %q", got, encodingGzip)
	}

	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v, want nil", err)
	}

	decoded, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip decode error = %v, want nil", err)
	}

	wantAttr := `nonce="` + cspNonce + `"`

	if !strings.Contains(string(decoded), wantAttr) {
		t.Errorf(
			"decompressed body does not embed the CSP nonce %q: body %q",
			wantAttr,
			string(decoded),
		)
	}
}

// TestChain_NonceThenCORS_CSPAndAllowOriginCoexist verifies the nonce and CORS
// middlewares do not overwrite each other's response headers when composed.
func TestChain_NonceThenCORS_CSPAndAllowOriginCoexist(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Chain(inner, CORS(DefaultCORSConfig()), Nonce(DefaultNonceConfig()))

	req := newTestRequest(http.MethodGet, "/", "https://example.com")

	rec := newRecorder()
	wrapped.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}

	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("Content-Security-Policy header = empty, want the nonce-bearing CSP")
	}
}

// TestChain_NonceWithServerTiming_NonceSurvivesTimingWrapper verifies a
// handler that wraps its writer with WrapServerTiming still sees the nonce in
// the request context, and both the Server-Timing and CSP headers land on the
// response.
func TestChain_NonceWithServerTiming_NonceSurvivesTimingWrapper(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timingWriter, timingReq := servertiming.WrapServerTiming(w, r)

		_, _ = fmt.Fprintf(timingWriter, "nonce=%s", NonceFromRequest(timingReq))
	})

	wrapped := Chain(inner, Nonce(DefaultNonceConfig()))

	req := newTestRequest(http.MethodGet, "/", "")

	rec := newRecorder()
	wrapped.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header = empty, want the nonce-bearing CSP")
	}

	if got := rec.Header().Get(servertiming.HeaderServerTiming); got == "" {
		t.Error("Server-Timing header = empty, want non-empty from the timing wrapper")
	}

	want := "nonce=" + cspNonceFromHeader(t, csp)

	if rec.Body.String() != want {
		t.Errorf(
			"body = %q, want %q (context nonce must match the CSP nonce)",
			rec.Body.String(),
			want,
		)
	}
}

// TestChain_NonceWithCSRFTokenHelpers_BothAttributesRender verifies a handler
// can render CSRF token helpers and the nonce attribute from one request: the
// nonce and CSRF token context values coexist without key collisions.
func TestChain_NonceWithCSRFTokenHelpers_BothAttributesRender(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithCSRFToken(r.Context(), "test-csrf-token")

		_, _ = fmt.Fprintf(
			w,
			"%s %s",
			CSRFTokenFormField(r.WithContext(ctx)),
			NonceAttr(r.WithContext(ctx)),
		)
	})

	wrapped := Chain(inner, Nonce(DefaultNonceConfig()))

	req := newTestRequest(http.MethodGet, "/", "")

	rec := newRecorder()
	wrapped.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `name="csrf_token" value="test-csrf-token"`) {
		t.Errorf("body %q missing the CSRF form field with the context token", body)
	}

	csp := rec.Header().Get("Content-Security-Policy")

	wantAttr := `nonce="` + cspNonceFromHeader(t, csp) + `"`

	if !strings.Contains(body, wantAttr) {
		t.Errorf("body %q missing the nonce attribute %q", body, wantAttr)
	}
}
