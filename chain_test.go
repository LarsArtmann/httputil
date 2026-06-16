package httputil

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

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

	assertStatus(t, rec, http.StatusOK)

	assertHeader(t, rec, headerContentEncoding, encodingGzip)

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

	assertStatus(t, rec, http.StatusNotModified)

	assertBodyEmpty(t, rec, "for 304")
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

	assertStatus(t, rec, http.StatusOK)

	// Still has both headers, but ETag is computed on compressed body.
	assertHeader(t, rec, headerContentEncoding, encodingGzip)

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

	inner := newWriteStatusHandler(http.StatusOK, "")

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
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	handler := Chain(inner, Compression(compCfg), ETag(etagCfg))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	// Small response: compression skipped, Content-Length should remain.
	if rec.Header().Get("Content-Length") != strconv.Itoa(len(body)) {
		t.Errorf(
			"Content-Length = %q, want %q for uncompressed response",
			rec.Header().Get("Content-Length"),
			strconv.Itoa(len(body)),
		)
	}

	// Compression should NOT have been applied.
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Content-Encoding should not be gzip for small response")
	}
}

func BenchmarkChain(b *testing.B) {
	logger := newTestLogger()
	corsCfg := DefaultCORSConfig()
	reqCfg := DefaultRequestIDConfig()
	secCfg := DefaultSecurityHeadersConfig()

	inner := newWriteStatusHandler(http.StatusOK, "")

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
