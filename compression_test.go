package httputil

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestCompression_NoAcceptEncoding(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteStatusHandler(http.StatusOK, "hello"))

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
	handler := Compression(cfg)(newWriteStatusHandler(http.StatusOK, "small"))

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

func testCompressionSkipsContentType(t *testing.T, contentType, wantErrMsg string) {
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
		t.Errorf("Content-Encoding = %q, want empty for %s", got, wantErrMsg)
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

	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

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

	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

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

	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty for image/png with default skip list", got)
	}
}
