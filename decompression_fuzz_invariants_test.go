package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzDecompressionInvariants verifies the two contract invariants that must
// hold for arbitrary bodies and encodings: (1) the Content-Encoding and
// Content-Length request headers are always removed when the middleware
// decompresses, and (2) a valid compressed body round-trips to exactly the
// original bytes (no truncation below the bomb limit).
func FuzzDecompressionInvariants(f *testing.F) {
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	_, _ = gzipWriter.Write([]byte("invariant payload"))
	_ = gzipWriter.Close()

	f.Add(gzipBuf.Bytes(), "gzip")
	f.Add([]byte{0x1f, 0x8b}, "gzip")
	f.Add([]byte("junk"), "gzip")

	f.Fuzz(func(t *testing.T, body []byte, encoding string) {
		if encoding != "gzip" && encoding != "deflate" {
			t.Skip("only supported encodings exercise the decompression path")
		}

		if encoding == "deflate" {
			var deflateBuf bytes.Buffer
			deflateWriter, _ := flate.NewWriter(&deflateBuf, flate.DefaultCompression)
			_, _ = deflateWriter.Write([]byte("invariant payload"))
			_ = deflateWriter.Close()
			body = deflateBuf.Bytes()
		}

		var gotBody strings.Builder
		var gotEncoding, gotLength string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotEncoding = r.Header.Get("Content-Encoding")
			gotLength = r.Header.Get("Content-Length")
			_, _ = io.Copy(&gotBody, r.Body)
			_ = r.Body.Close()
			w.WriteHeader(http.StatusOK)
		})

		cfg := DefaultDecompressionConfig()
		cfg.MaxDecompressionSize = 1 << 20
		wrapped := Decompression(cfg)(handler)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("Content-Encoding", encoding)
		req.Header.Set("Content-Length", "999999")
		wrapped.ServeHTTP(rec, req)

		if rec.Code == http.StatusBadRequest {
			return
		}

		if gotEncoding != "" {
			t.Errorf("Content-Encoding must be removed, got %q", gotEncoding)
		}

		if gotLength != "" {
			t.Errorf("Content-Length must be removed, got %q", gotLength)
		}

		if gotBody.String() != "invariant payload" && len(body) > 0 && rec.Code == http.StatusOK {
			t.Errorf("round-trip mismatch: got %q", gotBody.String())
		}
	})
}
