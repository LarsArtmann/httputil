package httputil

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultSecurityHeadersConfig()
	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	tests := []struct{ header, want string }{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		if got := rec.Header().Get(tt.header); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestSecurityHeaders_CustomCSP(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'",
	}
	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "Content-Security-Policy", "default-src 'self'")
}

func TestRequestID_GeneratesID(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()
	handler := RequestID(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("X-Request-ID")
	if got == "" {
		t.Error("X-Request-ID header is empty, want generated ID")
	}
}

func TestRequestID_ForwardsExistingID(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()

	var ctxID string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
	})

	handler := RequestID(cfg)(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("X-Request-ID", "existing-id-123")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "X-Request-ID", "existing-id-123")

	if ctxID != "existing-id-123" {
		t.Errorf("context ID = %q, want %q", ctxID, "existing-id-123")
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRecovery_PassesThroughNormal(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	called := false

	handler := Recovery(logger)(newCountingHandler(&called))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
}

func TestTimeout_SetsDeadline(t *testing.T) {
	t.Parallel()

	handler := Timeout(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("context has no deadline")
		}

		if deadline.IsZero() {
			t.Error("deadline is zero")
		}
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)
}

func TestLogging_RecordsRequest(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := Logging(logger)(inner)

	req := newTestRequest(http.MethodGet, "/test", "")
	req.RemoteAddr = "10.0.0.1:1234"

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if buf.Len() == 0 {
		t.Error("expected log output")
	}
}

func TestChain_ETagThenCompression_CorrectOrder(t *testing.T) {
	t.Parallel()

	etagCfg := DefaultETagConfig()
	compressCfg := DefaultCompressionConfig()

	body := []byte(strings.Repeat("a", defaultCompressionMinSize*2))

	inner := newWriteBodyHandler(body)

	// ETag should be inner (sees uncompressed body), Compression outer.
	// Chain applies in reverse: first = outermost.
	handler := Chain(inner, Compression(compressCfg), ETag(etagCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get(headerContentEncoding); got != encodingGzip {
		t.Errorf("Content-Encoding = %q, want %q", got, encodingGzip)
	}

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag")
	}
}

func TestChain_ETagThenCompression_IfNoneMatch304(t *testing.T) {
	t.Parallel()

	etagCfg := DefaultETagConfig()
	compressCfg := DefaultCompressionConfig()

	body := []byte(strings.Repeat("a", defaultCompressionMinSize*2))

	inner := newWriteBodyHandler(body)

	handler := Chain(inner, Compression(compressCfg), ETag(etagCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)
	req.Header.Set(headerIfNoneMatch, `"7c5597b9"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body length = %d, want 0 for 304", rec.Body.Len())
	}
}

func TestChain_CompressionThenETag_WrongOrder(t *testing.T) {
	t.Parallel()

	etagCfg := DefaultETagConfig()
	compressCfg := DefaultCompressionConfig()

	body := []byte(strings.Repeat("a", defaultCompressionMinSize*2))

	inner := newWriteBodyHandler(body)

	// WRONG order: Compression inner, ETag outer.
	// ETag sees compressed bytes, so ETag changes on every request
	// (gzip includes timestamp/metadata).
	handler := Chain(inner, ETag(etagCfg), Compression(compressCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Still has both headers, but ETag is computed on compressed body.
	if got := rec.Header().Get(headerContentEncoding); got != encodingGzip {
		t.Errorf("Content-Encoding = %q, want %q", got, encodingGzip)
	}

	if got := rec.Header().Get(headerETag); got == "" {
		t.Error("ETag header is empty")
	}
}

func TestChain_RecoveryLoggingCORS(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()

	inner := newWriteStatusHandler(http.StatusOK, "ok")

	handler := Chain(inner, CORS(corsCfg), Recovery(logger), Logging(logger))

	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	assertHeader(t, rec, "Access-Control-Allow-Origin", "*")
}

func TestChain_RecoveryCatchesPanicWithLogging(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("integration test panic")
	})

	handler := Chain(inner, CORS(corsCfg), Recovery(logger), Logging(logger))

	req := newTestRequest(http.MethodGet, "/", "http://example.com")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestChain_RequestIDSecurityHeaders(t *testing.T) {
	t.Parallel()

	reqCfg := DefaultRequestIDConfig()
	secCfg := DefaultSecurityHeadersConfig()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func BenchmarkRequestID(b *testing.B) {
	cfg := DefaultRequestIDConfig()
	middleware := RequestID(cfg)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkSecurityHeaders(b *testing.B) {
	cfg := DefaultSecurityHeadersConfig()
	middleware := SecurityHeaders(cfg)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkRecovery(b *testing.B) {
	logger := newTestLogger()
	middleware := Recovery(logger)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkTimeout(b *testing.B) {
	middleware := Timeout(time.Second)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkLogging(b *testing.B) {
	logger := newTestLogger()
	middleware := Logging(logger)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkChain(b *testing.B) {
	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()
	reqCfg := DefaultRequestIDConfig()
	secCfg := DefaultSecurityHeadersConfig()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

func FuzzRequestID(f *testing.F) {
	f.Add("existing-id-123")
	f.Add("")
	f.Add("a")
	f.Add("x-request-id-with-dashes-and-numbers-42")

	f.Fuzz(func(t *testing.T, headerValue string) {
		cfg := DefaultRequestIDConfig()
		handler := RequestID(cfg)(newNoOpHandler())

		req := newTestRequest(http.MethodGet, "/", "")
		req.Header.Set(cfg.ForwardHeader, headerValue)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		got := rec.Header().Get(cfg.HeaderName)
		if got == "" {
			t.Error("request ID header is empty")
		}
	})
}

func TestRequestIDConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestRequestIDConfig_Validate_NilGenerateID(t *testing.T) {
	t.Parallel()

	cfg := RequestIDConfig{
		HeaderName:    "X-Request-ID",
		ForwardHeader: "X-Request-ID",
		GenerateID:    nil,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for nil GenerateID")
	}

	if !errors.Is(err, errNilGenerateID) {
		t.Errorf("Validate() error = %v, want errNilGenerateID", err)
	}
}

func TestRequestIDConfig_Validate_EmptyHeaderName(t *testing.T) {
	t.Parallel()

	cfg := RequestIDConfig{
		HeaderName:    "",
		ForwardHeader: "X-Request-ID",
		GenerateID:    func() string { return "id" },
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty HeaderName")
	}

	if !errors.Is(err, errEmptyHeaderName) {
		t.Errorf("Validate() error = %v, want errEmptyHeaderName", err)
	}
}

func TestRequestIDConfig_Validate_EmptyForwardHeader(t *testing.T) {
	t.Parallel()

	cfg := RequestIDConfig{
		HeaderName:    "X-Request-ID",
		ForwardHeader: "",
		GenerateID:    func() string { return "id" },
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty ForwardHeader")
	}

	if !errors.Is(err, errEmptyForwardHdr) {
		t.Errorf("Validate() error = %v, want errEmptyForwardHdr", err)
	}
}

func TestSecurityHeadersConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultSecurityHeadersConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestSecurityHeadersConfig_Validate_EmptyConfig(t *testing.T) {
	t.Parallel()

	cfg := SecurityHeadersConfig{}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for empty config", err)
	}
}

func TestChain_CompressionETag_HijackPassthrough(t *testing.T) {
	t.Parallel()

	compCfg := DefaultCompressionConfig()
	etagCfg := DefaultETagConfig()

	var hijackCalled bool

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("inner handler: ResponseWriter does not implement Hijacker")
		}

		_, _, err := hijacker.Hijack()
		if err != nil {
			t.Fatalf("Hijack() error: %v", err)
		}

		hijackCalled = true
	})

	handler := Chain(inner, Compression(compCfg), ETag(etagCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := newHijackRecorder()

	handler.ServeHTTP(rec, req)

	if !hijackCalled {
		t.Error("Hijack was not called through Compression + ETag chain")
	}

	if !rec.hijacked {
		t.Error("underlying Hijack was not called")
	}
}

func TestChain_CompressionETag_SmallResponsePreservesContentLength(t *testing.T) {
	t.Parallel()

	compCfg := DefaultCompressionConfig()
	etagCfg := DefaultETagConfig()

	body := []byte("small response under min size")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	handler := Chain(inner, Compression(compCfg), ETag(etagCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Small response: compression skipped, Content-Length should remain.
	if rec.Header().Get("Content-Length") != itoa(len(body)) {
		t.Errorf(
			"Content-Length = %q, want %q for uncompressed response",
			rec.Header().Get("Content-Length"),
			itoa(len(body)),
		)
	}

	// Compression should NOT have been applied.
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Content-Encoding should not be gzip for small response")
	}
}
