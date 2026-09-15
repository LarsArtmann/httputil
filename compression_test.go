package httputil

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestCompression_NoAcceptEncoding(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteStatusHandler("hello"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	assertHeader(t, rec, headerContentEncoding, "")

	assertBody(t, rec, "hello")
}

func TestCompression_AcceptEncoding_Gzip(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	assertHeader(t, rec, headerContentEncoding, encodingGzip)

	gzipReader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader error = %v", err)
	}

	defer func() { _ = gzipReader.Close() }()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if len(decompressed) != defaultCompressionMinSize+1 {
		t.Errorf(
			"decompressed length = %d, want %d",
			len(decompressed),
			defaultCompressionMinSize+1,
		)
	}
}

func TestCompression_SmallResponse(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteStatusHandler("small"))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, headerContentEncoding, "")

	assertBody(t, rec, "small")
}

func TestCompression_Non2xxStatus(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(strings.Repeat("x", defaultCompressionMinSize+1)))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty for non-2xx", got)
	}
}

func TestCompression_AlreadyEncoded(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerContentEncoding, "br")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, headerContentEncoding, "br")
}

func TestCompression_Flush(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newFlushHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty after flush", got)
	}

	assertBody(t, rec, "partial more")
}

func TestCompression_EmptyResponse(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty for empty response", got)
	}
}

func TestCompression_VaryHeader(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newNoOpHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	vary := rec.Header().Values(headerVary)
	found := slices.Contains(vary, headerAcceptEncoding)

	if !found {
		t.Errorf("Vary header missing %q, got %v", headerAcceptEncoding, vary)
	}
}

func TestCompressionConfig_Validate_Valid(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestCompressionConfig_Validate_InvalidLevel(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{MinSize: 100, Level: 99}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for invalid level")
	}

	if !strings.Contains(err.Error(), "compression level") {
		t.Errorf("error = %v, want compression level error", err)
	}
}

func testCompressionSkipsContentType(t *testing.T, contentType, label string) {
	t.Helper()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty for %s", got, label)
	}
}

func TestCompression_SkipsImageContentType(t *testing.T) {
	t.Parallel()

	testCompressionSkipsContentType(t, "image/png", "image/png")
}

func TestCompression_SkipsVideoContentType(t *testing.T) {
	t.Parallel()

	testCompressionSkipsContentType(t, "video/mp4", "video/mp4")
}

func TestCompression_SkipsGzipContentType(t *testing.T) {
	t.Parallel()

	testCompressionSkipsContentType(t, "application/gzip", "application/gzip")
}

func TestCompression_Hijack_SetsPlainMode(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	compressWriter := newCompressWriter(
		newHijackRecorder(),
		cfg.MinSize,
		encodingGzip,
		GzipWriterFactory(cfg.Level),
		newWriterPool(GzipWriterFactory(cfg.Level)),
		nil,
	)

	_, _, err := compressWriter.Hijack()
	if err != nil {
		t.Fatalf("Hijack() error = %v, want nil", err)
	}

	if !compressWriter.plain {
		t.Error("Hijack should set plain mode")
	}
}

func TestCompressionConfig_Validate_NegativeMinSize(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{MinSize: -1, Level: gzip.DefaultCompression}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative MinSize")
	}

	if !strings.Contains(err.Error(), "minimum size") {
		t.Errorf("error = %v, want minimum size error", err)
	}
}

func TestCompression_WriteCompressedPath(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{MinSize: 1, Level: gzip.DefaultCompression}
	body := strings.Repeat("a", 300)

	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
		_, _ = w.Write([]byte("second write"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Content-Encoding is not gzip")
	}
}

func TestCompression_FlushWhileBuffering(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{MinSize: 1000, Level: gzip.DefaultCompression}

	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("small"))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress when flushed below min size")
	}
}

func TestCompression_CustomIncompressibleTypes(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.IncompressibleTypes = []string{"text/"}

	handler := Compression(
		cfg,
	)(
		newTypedBodyHandler("text/plain", strings.Repeat("a", defaultCompressionMinSize+1)),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty for custom-skipped text/plain", got)
	}
}

func TestCompression_EmptyIncompressibleTypesCompressesAll(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.IncompressibleTypes = []string{}

	handler := Compression(
		cfg,
	)(
		newTypedBodyHandler("image/png", strings.Repeat("a", defaultCompressionMinSize+1)),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != encodingGzip {
		t.Errorf("Content-Encoding = %q, want %q for empty skip list", got, encodingGzip)
	}
}

func TestCompression_NilIncompressibleTypesUsesDefaults(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.IncompressibleTypes = nil

	handler := Compression(
		cfg,
	)(
		newTypedBodyHandler("image/png", strings.Repeat("a", defaultCompressionMinSize+1)),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty for image/png with default skip list", got)
	}
}

// simpleWriteCloser implements io.WriteCloser but NOT Reset or Flush,
// exercising the fresh-writer fallback and nopFlushCloser wrapper in
// startCompression.
type simpleWriteCloser struct {
	dst io.Writer
}

func (w *simpleWriteCloser) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	if err != nil {
		return n, fmt.Errorf("simpleWriteCloser: %w", err)
	}

	return n, nil
}

func (w *simpleWriteCloser) Close() error {
	return nil
}

func TestCompression_CustomFactoryWithoutReset(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.WriterFactories = map[string]WriterFactory{
		"gzip":     GzipWriterFactory(cfg.Level),
		"deflate":  DeflateWriterFactory(cfg.Level),
		"identity": passthroughFactory,
		"custom": func(dst io.Writer) (io.WriteCloser, error) {
			return &simpleWriteCloser{dst: dst}, nil
		},
	}

	handler := Compression(cfg)(
		newTypedBodyHandler("text/plain", strings.Repeat("a", defaultCompressionMinSize+1)),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, "custom")

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, headerContentEncoding, "custom")

	if rec.Body.Len() != defaultCompressionMinSize+1 {
		t.Errorf(
			"body length = %d, want %d (passthrough, no compression)",
			rec.Body.Len(),
			defaultCompressionMinSize+1,
		)
	}
}

// TestCompression_FlushWhileCompressing exercises the compressing branch of
// compressWriter.Flush: a response large enough to start compression is
// written, then Flush is called mid-stream. This drives the w.writer.Flush() +
// responseWrapper.Flush() path that the existing FlushWhileBuffering test
// (which flushes below the threshold) does not reach.
func TestCompression_FlushWhileCompressing(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{MinSize: 1, Level: gzip.DefaultCompression}
	body := strings.Repeat("a", defaultCompressionMinSize+1)

	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, headerContentEncoding, encodingGzip)
}

// TestCompression_FlushNonFlushableCustomWriter exercises the nopFlushCloser
// wrapper: a custom factory writer without a Flush method is wrapped in
// nopFlushCloser by startCompression, and a mid-stream Flush calls
// nopFlushCloser.Flush (the no-op path for encodings that cannot flush).
func TestCompression_FlushNonFlushableCustomWriter(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.WriterFactories = map[string]WriterFactory{
		"gzip":     GzipWriterFactory(cfg.Level),
		"deflate":  DeflateWriterFactory(cfg.Level),
		"identity": passthroughFactory,
		"custom": func(dst io.Writer) (io.WriteCloser, error) {
			return &simpleWriteCloser{dst: dst}, nil
		},
	}

	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, "custom")

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, headerContentEncoding, "custom")
}

// TestCompression_InvalidConfigContinues verifies that an invalid
// compression config (negative MinSize) is logged by the constructor (via the
// shared validateConfig helper) but does not prevent the middleware from
// constructing and serving requests. The log emission itself is covered by
// validate_config_log_test.go.
func TestCompression_InvalidConfigContinues(t *testing.T) {
	t.Parallel()

	// MinSize < 0 is always a bug. The constructor fills WriterFactories from
	// defaults first, then Validate returns errNegativeMinSize and logs it.
	cfg := CompressionConfig{
		MinSize: -1,
	}

	var called bool

	handler := Compression(cfg)(newCountingHandler(&called))
	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called (invalid config should log and continue)")
	}
}

func TestCompression_ZeroLevelMeansDefaultCompression(t *testing.T) {
	t.Parallel()

	const bodySize = 10_000

	payload := bytes.Repeat([]byte("a"), bodySize)

	handler := Compression(
		CompressionConfig{},
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(headerContentType, "text/plain")
			_, _ = w.Write(payload)
		}),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	assertHeader(t, rec, headerContentEncoding, encodingGzip)

	gzipReader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader error = %v", err)
	}

	defer func() { _ = gzipReader.Close() }()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if !bytes.Equal(decompressed, payload) {
		t.Error("gunzipped body should round-trip the original payload")
	}

	if rec.Body.Len() >= bodySize {
		t.Errorf(
			"compressed length = %d, want well below %d (Level == 0 must select gzip.DefaultCompression, not gzip.NoCompression stored blocks)",
			rec.Body.Len(),
			bodySize,
		)
	}
}

func TestCompressionConfig_Validate_ZeroLevelIsValid(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{Level: 0, MinSize: 1, WriterFactories: DefaultWriterFactories()}

	if err := cfg.Validate(); err != nil {
		t.Fatalf(
			`Validate error = %v, want nil (Level == 0 means "unset, default compression")`,
			err,
		)
	}
}

// TestCompression_ExactMinSizeWrite_IsNotDuplicated pins the exact-fill
// duplication regression through the full middleware: a handler Write that
// exactly reaches MinSize must produce a decoded stream equal to the payload
// (the compressor consumes the buffered bytes instead of re-emitting them).
func TestCompression_ExactMinSizeWrite_IsNotDuplicated(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte("x"), defaultCompressionMinSize)

	handler := Compression(
		DefaultCompressionConfig(),
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(headerContentType, "text/plain")
			_, _ = w.Write(payload)
		}),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	assertHeader(t, rec, headerContentEncoding, encodingGzip)

	gzReader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader error = %v", err)
	}

	defer func() { _ = gzReader.Close() }()

	decoded, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("gzip decode error = %v", err)
	}

	if !bytes.Equal(decoded, payload) {
		t.Errorf(
			"decoded body = %d bytes, want %d (exact-fill must not duplicate payload)",
			len(decoded),
			len(payload),
		)
	}
}

func TestCompressionConfig_Validate_SkipsLevelCheckWhenFactoriesSet(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.Level = 99

	if err := cfg.Validate(); err != nil {
		t.Fatalf(
			"Validate() error = %v, want nil (Level is ignored when WriterFactories is set)",
			err,
		)
	}
}

func TestCompressionConfig_Validate_RejectsInvalidIncompressiblePrefix(t *testing.T) {
	t.Parallel()

	for _, prefix := range []string{"", "image", " image/"} {
		cfg := DefaultCompressionConfig()
		cfg.IncompressibleTypes = []string{prefix}

		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() error = nil, want error for invalid prefix %q", prefix)
		} else if !errors.Is(err, errIncompressiblePrefixInvalid) {
			t.Errorf(
				"Validate() error = %v, want errIncompressiblePrefixInvalid for prefix %q",
				err,
				prefix,
			)
		}
	}
}

func TestCompressionConfig_Validate_AcceptsDefaultIncompressibleTypes(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil for DefaultIncompressibleTypes", err)
	}
}

// TestCompressionConfig_Validate_RejectsUnknownAbsentEncodingPolicy specifies
// that only the defined AbsentEncodingPolicy constants pass validation; an
// out-of-range value is a config Rejection, not a silently-defaulted field.
func TestCompressionConfig_Validate_RejectsUnknownAbsentEncodingPolicy(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.AbsentEncoding = AbsentEncodingPolicy(42)

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want errAbsentEncodingInvalid for policy 42")
	} else if !errors.Is(err, errAbsentEncodingInvalid) {
		t.Errorf("Validate() error = %v, want errAbsentEncodingInvalid for policy 42", err)
	}
}

// TestCompressionConfig_Validate_AcceptsBothAbsentEncodingPolicies specifies
// that both defined AbsentEncodingPolicy constants pass validation.
func TestCompressionConfig_Validate_AcceptsBothAbsentEncodingPolicies(t *testing.T) {
	t.Parallel()

	for _, policy := range []AbsentEncodingPolicy{AbsentEncodingIdentity, AbsentEncodingFirstConfigured} {
		cfg := DefaultCompressionConfig()
		cfg.AbsentEncoding = policy

		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil for policy %d", err, policy)
		}
	}
}

// TestCompression_ZeroValueAbsentEncodingServesUncompressed pins the
// zero-value contract of CompressionConfig.AbsentEncoding ("0 means
// identity", issue #4): a config literal that leaves the field unset serves
// the uncompressed representation to requests without an Accept-Encoding
// header.
func TestCompression_ZeroValueAbsentEncodingServesUncompressed(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{
		MinSize:             0,
		WriterFactories:     DefaultWriterFactories(),
		IncompressibleTypes: nil,
	}

	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertUncompressedResponse(t, rec, strings.Repeat("a", defaultCompressionMinSize+1))
}
