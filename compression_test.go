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

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty", got)
	}

	if got := rec.Body.String(); got != "hello" {
		t.Errorf("body = %q, want %q", got, "hello")
	}
}

func TestCompression_AcceptEncoding_Gzip(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

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

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want empty", got)
	}

	if got := rec.Body.String(); got != "small" {
		t.Errorf("body = %q, want %q", got, "small")
	}
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

	if got := rec.Body.String(); got != "partial more" {
		t.Errorf("body = %q, want %q", got, "partial more")
	}
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
	compressWriter := newCompressWriter(newHijackRecorder(), cfg.MinSize, cfg.Level)

	_, _, err := compressWriter.Hijack()
	if err != nil {
		t.Fatalf("Hijack() error = %v, want nil", err)
	}

	if !compressWriter.plain {
		t.Error("Hijack should set plain mode")
	}
}

func TestCompression_Push_Delegates(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	compressWriter := newCompressWriter(newPushRecorder(), cfg.MinSize, cfg.Level)

	err := compressWriter.Push("/test", nil)
	if err != nil {
		t.Errorf("Push error = %v, want nil", err)
	}
}

func FuzzCompression(f *testing.F) {
	f.Add([]byte("hello world"), "gzip")
	f.Add([]byte(strings.Repeat("a", 1024)), "gzip, deflate")
	f.Add([]byte(""), "")

	cfg := DefaultCompressionConfig()
	inner := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte("response body"))
	})

	f.Fuzz(func(t *testing.T, body []byte, acceptEncoding string) {
		handler := Compression(cfg)(inner)

		req := newTestRequest(http.MethodGet, "/", "")
		req.Header.Set(headerAcceptEncoding, acceptEncoding)

		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func BenchmarkCompression(b *testing.B) {
	cfg := DefaultCompressionConfig()
	middleware := Compression(cfg)

	body := []byte(strings.Repeat("a", defaultCompressionMinSize*2))

	inner := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write(body)
	})

	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
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

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

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

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress when flushed below min size")
	}
}
