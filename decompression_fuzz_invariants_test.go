package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzDecompressionInvariants verifies the two contract invariants that must
// hold for arbitrary payloads and supported encodings: (1) the Content-Encoding
// and Content-Length request headers are always removed when the middleware
// decompresses, and (2) a body that survives decompression round-trips to
// exactly the fuzzer-chosen plaintext, verified against an independent
// reference decoder (not a fixed payload). Corruption coverage lives in
// FuzzDecompression, which feeds raw unencoded bytes; this target always
// constructs valid wire streams so the round-trip assertion is sound.
func FuzzDecompressionInvariants(f *testing.F) {
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	_, _ = gzipWriter.Write([]byte("invariant payload"))
	_ = gzipWriter.Close()

	f.Add([]byte("invariant payload"), "gzip")
	f.Add([]byte(""), "gzip")
	f.Add([]byte(""), "deflate")
	f.Add(bytes.Repeat([]byte{0xAA}, 4096), "deflate")

	f.Fuzz(func(t *testing.T, body []byte, encoding string) {
		if encoding != "gzip" && encoding != "deflate" {
			t.Skip("only supported encodings exercise the decompression path")
		}

		var wire bytes.Buffer

		const maxWireBytes = 1 << 20

		if encoding == "gzip" {
			gzipEncoder := gzip.NewWriter(&wire)
			_, _ = gzipEncoder.Write(body)
			_ = gzipEncoder.Close()
		} else {
			flateEncoder, err := flate.NewWriter(&wire, flate.DefaultCompression)
			if err != nil {
				t.Skip("flate.NewWriter failed")
			}

			_, _ = flateEncoder.Write(body)
			_ = flateEncoder.Close()
		}

		var gotBody bytes.Buffer

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
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(wire.Bytes()))
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

		var want bytes.Buffer

		// Bound the reference decode so the harness itself cannot be turned
		// into a decompression bomb; the middleware under test is bound by
		// MaxDecompressionSize, so a larger reference is unreachable anyway.
		if encoding == "gzip" {
			refReader, err := gzip.NewReader(bytes.NewReader(wire.Bytes()))
			if err != nil {
				return
			}

			if _, err := io.Copy(&want, io.LimitReader(refReader, maxWireBytes+1)); err != nil {
				return
			}
		} else {
			refReader := flate.NewReader(bytes.NewReader(wire.Bytes()))

			if _, err := io.Copy(&want, io.LimitReader(refReader, maxWireBytes+1)); err != nil {
				return
			}
		}

		if gotBody.String() != want.String() {
			t.Errorf(
				"round-trip mismatch: middleware emitted %d bytes, reference decode is %d bytes",
				gotBody.Len(),
				want.Len(),
			)
		}
	})
}
