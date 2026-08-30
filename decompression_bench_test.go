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

const (
	benchDecompressionPayloadSize = 16 * 1024 // 16 KiB
	benchDecompressionSeedText    = "the quick brown fox jumps over the lazy dog"
)

func benchDecompressionPayload(b *testing.B, encoding string) []byte {
	b.Helper()

	repetitions := benchDecompressionPayloadSize/len(benchDecompressionSeedText) + 1
	payload := bytes.Repeat([]byte(benchDecompressionSeedText), repetitions)
	payload = payload[:benchDecompressionPayloadSize]

	var buf bytes.Buffer

	switch encoding {
	case encodingGzip:
		writer := gzip.NewWriter(&buf)
		_, err := writer.Write(payload)
		if err != nil {
			b.Fatalf("gzip write: %v", err)
		}

		err = writer.Close()
		if err != nil {
			b.Fatalf("gzip close: %v", err)
		}
	case encodingDeflate:
		writer, err := flate.NewWriter(&buf, flate.DefaultCompression)
		if err != nil {
			b.Fatalf("flate writer: %v", err)
		}

		_, err = writer.Write(payload)
		if err != nil {
			b.Fatalf("flate write: %v", err)
		}

		err = writer.Close()
		if err != nil {
			b.Fatalf("flate close: %v", err)
		}
	default:
		b.Fatalf("unknown encoding: %s", encoding)
	}

	return buf.Bytes()
}

// BenchmarkDecompression measures request-body decompression throughput as
// b.Run sub-benchmarks (modernized from three top-level benchmarks, 08-52:f49):
//
//   - gzip/deflate: full decompression path, including the limitedReader
//     bomb-protection wrapper.
//   - passthrough: no Content-Encoding set; the middleware short-circuits,
//     giving the per-request floor cost.
//
// Both the body reader and the Content-Encoding header are restored every
// iteration: Decompression consumes the body and DELETES the header from the
// request it serves, so a naively reused request silently degrades the
// benchmark to the passthrough path after the first iteration (the three
// top-level benchmarks this replaces had exactly that bug — their baseline
// numbers measured ~1 decompression amortized over millions of no-ops).
func BenchmarkDecompression(b *testing.B) {
	compressed := map[string][]byte{
		encodingGzip:    benchDecompressionPayload(b, encodingGzip),
		encodingDeflate: benchDecompressionPayload(b, encodingDeflate),
	}

	uncompressed := []byte("uncompressed request body")

	handler := Decompression(
		DefaultDecompressionConfig(),
	)(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
		}),
	)

	b.Run("gzip", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)

		b.ReportAllocs()
		b.SetBytes(benchDecompressionPayloadSize)

		for b.Loop() {
			req.Header.Set(headerContentEncoding, encodingGzip)
			req.Body = io.NopCloser(bytes.NewReader(compressed[encodingGzip]))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("deflate", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)

		b.ReportAllocs()
		b.SetBytes(benchDecompressionPayloadSize)

		for b.Loop() {
			req.Header.Set(headerContentEncoding, encodingDeflate)
			req.Body = io.NopCloser(bytes.NewReader(compressed[encodingDeflate]))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})

	b.Run("passthrough", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)

		b.ReportAllocs()
		b.SetBytes(int64(len(uncompressed)))

		for b.Loop() {
			req.Body = io.NopCloser(bytes.NewReader(uncompressed))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}
