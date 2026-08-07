package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecompressionConfigValidateValid(t *testing.T) {
	t.Parallel()

	cfg := DefaultDecompressionConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestDecompressionConfigValidateNegative(t *testing.T) {
	t.Parallel()

	cfg := DecompressionConfig{
		Encodings:            []string{encodingGzip},
		MaxDecompressionSize: -1,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative MaxDecompressionSize")
	}

	if !errors.Is(err, errMaxDecompressionSizeNegative) {
		t.Errorf("Validate() error = %v, want errMaxDecompressionSizeNegative", err)
	}
}

func TestDecompressionGzip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("hello gzip"))
	_ = zw.Close()

	handler := Decompression(DefaultDecompressionConfig())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("read error"))

				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set(headerContentEncoding, encodingGzip)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "hello gzip" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "hello gzip")
	}
}

func TestDecompressionDeflate(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	zw, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	_, _ = zw.Write([]byte("hello deflate"))
	_ = zw.Close()

	handler := Decompression(DefaultDecompressionConfig())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("read error"))

				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set(headerContentEncoding, encodingDeflate)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "hello deflate" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "hello deflate")
	}
}

const (
	testBombProtectionLimit    = 128
	testBombProtectionOversize = 4096
)

// closeTrackingReadCloser is a minimal ReadCloser that records whether Close
// was called, used to verify limitedReadCloser delegates Close correctly.
type closeTrackingReadCloser struct {
	data   []byte
	closed bool
}

func (r *closeTrackingReadCloser) Read(p []byte) (int, error) {
	if r.closed || len(r.data) == 0 {
		return 0, io.EOF
	}

	n := copy(p, r.data)
	r.data = r.data[n:]

	return n, nil
}

func (r *closeTrackingReadCloser) Close() error {
	r.closed = true

	return nil
}

func TestLimitedReaderClose(t *testing.T) {
	t.Parallel()

	src := &closeTrackingReadCloser{data: []byte("test")}
	lrc := limitedReadCloser(src, defaultMaxDecompressionSize)

	err := lrc.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	if !src.closed {
		t.Error("Close() did not close the underlying reader")
	}
}

func TestLimitedReaderBombProtection(t *testing.T) {
	t.Parallel()

	data := bytes.Repeat([]byte{0}, testBombProtectionOversize)
	src := &closeTrackingReadCloser{data: data}
	lrc := limitedReadCloser(src, testBombProtectionLimit)

	body, err := io.ReadAll(lrc)
	if err == nil {
		t.Fatalf("ReadAll() error = nil, want error for exceeding bomb limit")
	}

	if !errors.Is(err, errDecompressionSizeExceeded) {
		t.Errorf("ReadAll() error = %v, want errDecompressionSizeExceeded", err)
	}

	if len(body) > testBombProtectionLimit {
		t.Errorf("read %d bytes, want at most %d", len(body), testBombProtectionLimit)
	}

	if !src.closed {
		t.Error("underlying reader was not closed when bomb limit exceeded")
	}
}

func TestDecompressionBombProtection(t *testing.T) {
	t.Parallel()

	var compressed bytes.Buffer

	zw := gzip.NewWriter(&compressed)
	payload := bytes.Repeat([]byte{0}, testBombProtectionOversize)

	_, _ = zw.Write(payload)
	_ = zw.Close()

	cfg := DecompressionConfig{
		Encodings:            []string{encodingGzip},
		MaxDecompressionSize: testBombProtectionLimit,
	}

	var handlerErr error

	handler := Decompression(cfg)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, handlerErr = io.ReadAll(r.Body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed.Bytes()))
	req.Header.Set(headerContentEncoding, encodingGzip)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if handlerErr == nil {
		t.Fatal("handler expected error for decompression bomb, got nil")
	}

	if !errors.Is(handlerErr, errDecompressionSizeExceeded) {
		t.Errorf("handler error = %v, want errDecompressionSizeExceeded", handlerErr)
	}
}

// errorReadCloser is a test ReadCloser with configurable Read and Close errors.
type errorReadCloser struct {
	readErr  error
	closeErr error
	closed   bool
}

func (r *errorReadCloser) Read(p []byte) (int, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}

	return 0, io.EOF
}

func (r *errorReadCloser) Close() error {
	r.closed = true

	return r.closeErr
}

var (
	errTestRead  = errors.New("test read failure")
	errTestClose = errors.New("test close failure")
)

func TestLimitedReaderCloseError(t *testing.T) {
	t.Parallel()

	src := &errorReadCloser{closeErr: errTestClose}
	lrc := limitedReadCloser(src, defaultMaxDecompressionSize)

	err := lrc.Close()
	if err == nil {
		t.Fatal("Close() error = nil, want error")
	}

	if !errors.Is(err, errTestClose) {
		t.Errorf("Close() error = %v, want errTestClose", err)
	}

	if !src.closed {
		t.Error("Close() did not call underlying Close")
	}
}

func TestLimitedReaderReadError(t *testing.T) {
	t.Parallel()

	src := &errorReadCloser{readErr: errTestRead}
	lrc := limitedReadCloser(src, defaultMaxDecompressionSize)

	_, err := lrc.Read(make([]byte, 16))
	if err == nil {
		t.Fatal("Read() error = nil, want error")
	}

	if !errors.Is(err, errTestRead) {
		t.Errorf("Read() error = %v, want errTestRead", err)
	}
}

func TestDecompressionPassesThroughUnencoded(t *testing.T) {
	t.Parallel()

	handler := Decompression(DefaultDecompressionConfig())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("plain text"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "plain text" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "plain text")
	}
}

func TestDecompressionRejectsInvalidGzip(t *testing.T) {
	t.Parallel()

	handler := Decompression(DefaultDecompressionConfig())(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not gzip data"))
	req.Header.Set(headerContentEncoding, encodingGzip)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecompressionRemovesEncodingHeaders(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("test"))
	_ = zw.Close()

	var seenEncoding, seenLength bool

	handler := Decompression(DefaultDecompressionConfig())(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seenEncoding = r.Header.Get(headerContentEncoding) != ""
			seenLength = r.Header.Get(headerContentLength) != ""
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set(headerContentEncoding, encodingGzip)
	req.Header.Set(headerContentLength, "100")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if seenEncoding {
		t.Error("Content-Encoding header should be removed")
	}

	if seenLength {
		t.Error("Content-Length header should be removed")
	}
}

func TestDecompressionRespectsEncodingFilter(t *testing.T) {
	t.Parallel()

	cfg := DecompressionConfig{
		Encodings:            []string{encodingGzip},
		MaxDecompressionSize: defaultMaxDecompressionSize,
	}

	var buf bytes.Buffer
	zw, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	_, _ = zw.Write([]byte("hello deflate"))
	_ = zw.Close()

	handler := Decompression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If deflate is not allowed, the body should pass through compressed
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(buf.Bytes()))
	req.Header.Set(headerContentEncoding, encodingDeflate)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Since deflate is not in the allowed list, the compressed data passes through
	if rec.Body.String() == "hello deflate" {
		t.Error("deflate should not have been decompressed when only gzip is allowed")
	}
}
