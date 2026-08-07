package httputil

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestChain_RecoveryLoggingCORS(t *testing.T) {
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

// TestChain_CompressionETag_Matching304 verifies that a 304 through the
// Compression+ETag chain carries the ETag header but excludes Content-Encoding.
func TestChain_CompressionETag_Matching304(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, `"known-etag"`)
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(strings.Repeat("hello world ", 100)))
	})

	handler := Chain(
		inner,
		Compression(DefaultCompressionConfig()),
		ETag(ETagConfig{SkipIfPresent: true}),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerIfNoneMatch, `"known-etag"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)

	if rec.Header().Get(headerETag) == "" {
		t.Error("304 response must include ETag header")
	}

	if ce := rec.Header().Get("Content-Encoding"); ce != "" {
		t.Errorf("304 response must not include Content-Encoding, got %q", ce)
	}
}

// TestChain_CompressionETag_NoMatch200 verifies that a 200 through the
// Compression+ETag chain compresses the body and includes the ETag header.
func TestChain_CompressionETag_NoMatch200(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("hello world ", 100)
	inner := newWriteStatusHandler(body)

	handler := Chain(inner, Compression(DefaultCompressionConfig()), ETag(DefaultETagConfig()))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	if rec.Header().Get(headerETag) == "" {
		t.Error("200 response must include ETag header through Compression+ETag chain")
	}

	assertHeader(t, rec, "Content-Encoding", "gzip")
}

// TestChain_CompressionETag_HijackPassthrough verifies that Hijack through
// the Compression+ETag chain delegates to the underlying writer.
func TestChain_CompressionETag_HijackPassthrough(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		hj, ok := rw.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Hijacker")
		}

		_, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("Hijack() error = %v, want nil", err)
		}
	})

	handler := Chain(
		inner,
		Compression(DefaultCompressionConfig()),
		ETag(DefaultETagConfig()),
	)

	rec := newHijackRecorder()
	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip")

	handler.ServeHTTP(rec, req)

	if !rec.hijacked {
		t.Error("underlying writer was not hijacked")
	}
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
