package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzDecompression verifies that the Decompression middleware never panics
// on arbitrary compressed bodies and encoding header values. The middleware
// must either successfully decompress valid data, reject malformed gzip with
// HTTP 400, or pass unrecognized encodings through to the handler.
func FuzzDecompression(f *testing.F) {
	// Valid gzip body.
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	_, _ = gzipWriter.Write([]byte("hello gzip fuzz"))
	_ = gzipWriter.Close()

	f.Add(gzipBuf.Bytes(), "gzip")
	f.Add([]byte("not gzip at all"), "gzip")
	f.Add([]byte{}, "gzip")
	f.Add([]byte{0x1f, 0x8b}, "gzip") // truncated gzip header
	f.Add(gzipBuf.Bytes(), "deflate")
	f.Add([]byte("raw bytes"), "deflate")
	f.Add(gzipBuf.Bytes(), "") // no encoding → passthrough
	f.Add([]byte("plain"), "identity")
	f.Add([]byte("plain"), "br") // unsupported encoding → passthrough
	f.Add([]byte{}, "")

	// Valid deflate body.
	var deflateBuf bytes.Buffer
	deflateWriter, _ := flate.NewWriter(&deflateBuf, flate.DefaultCompression)
	_, _ = deflateWriter.Write([]byte("hello deflate fuzz"))
	_ = deflateWriter.Close()
	f.Add(deflateBuf.Bytes(), "deflate")

	f.Fuzz(func(t *testing.T, body []byte, encoding string) {
		cfg := DefaultDecompressionConfig()
		handler := Decompression(
			cfg,
		)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Drain the body to trigger decompression errors (if any).
				//nolint:makezero // pre-allocated buffer for Read, not append
				drain := make([]byte, 1024)

				for {
					_, err := r.Body.Read(drain)
					if err != nil {
						break
					}
				}

				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		if encoding != "" {
			req.Header.Set(headerContentEncoding, encoding)
		}

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// The only acceptable outcomes are 200 (passthrough, successful
		// decompression, or a handler that ignores mid-stream read errors) and
		// 400 (gzip.NewReader rejects the body up front). Anything else is a
		// defect. Note the handler ignores read errors, so mid-stream corruption
		// legitimately ends in 200 — this target primarily guards against
		// panics and unexpected status codes.
		if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
			t.Errorf(
				"unexpected status %d for encoding %q, body len %d",
				rec.Code,
				encoding,
				len(body),
			)
		}
	})
}
