package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"
)

const benchDecompressionPayloadSize = 16 * 1024 // 16 KiB

func benchDecompressionPayload(b *testing.B, encoding string) []byte {
	b.Helper()

	payload := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog"), benchDecompressionPayloadSize/len("the quick brown fox jumps over the lazy dog")+1)
	payload = payload[:benchDecompressionPayloadSize]

	var buf bytes.Buffer

	switch encoding {
	case encodingGzip:
		zw := gzip.NewWriter(&buf)
		_, err := zw.Write(payload)
		if err != nil {
			b.Fatalf("gzip write: %v", err)
		}

		err = zw.Close()
		if err != nil {
			b.Fatalf("gzip close: %v", err)
		}
	case encodingDeflate:
		zw, err := flate.NewWriter(&buf, flate.DefaultCompression)
		if err != nil {
			b.Fatalf("flate writer: %v", err)
		}

		_, err = zw.Write(payload)
		if err != nil {
			b.Fatalf("flate write: %v", err)
		}

		err = zw.Close()
		if err != nil {
			b.Fatalf("flate close: %v", err)
		}
	default:
		b.Fatalf("unknown encoding: %s", encoding)
	}

	return buf.Bytes()
}

// BenchmarkDecompression_Gzip measures gzip request body decompression
// throughput, including the limitedReader bomb-protection wrapper.
func BenchmarkDecompression_Gzip(b *testing.B) {
	compressed := benchDecompressionPayload(b, encodingGzip)

	cfg := DefaultDecompressionConfig()
	handler := Decompression(cfg)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// Drain the decompressed body to simulate a real handler reading it.
		_, _ = readAllForBench(r)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed))
	req.Header.Set(headerContentEncoding, encodingGzip)

	b.ReportAllocs()
	b.SetBytes(benchDecompressionPayloadSize)

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkDecompression_Deflate measures deflate request body decompression
// throughput, including the limitedReader bomb-protection wrapper.
func BenchmarkDecompression_Deflate(b *testing.B) {
	compressed := benchDecompressionPayload(b, encodingDeflate)

	cfg := DefaultDecompressionConfig()
	handler := Decompression(cfg)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = readAllForBench(r)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed))
	req.Header.Set(headerContentEncoding, encodingDeflate)

	b.ReportAllocs()
	b.SetBytes(benchDecompressionPayloadSize)

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// BenchmarkDecompression_Passthrough measures the overhead when no
// Content-Encoding is set — the middleware should short-circuit immediately.
func BenchmarkDecompression_Passthrough(b *testing.B) {
	body := []byte("uncompressed request body")

	cfg := DefaultDecompressionConfig()
	handler := Decompression(cfg)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = readAllForBench(r)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

	b.ReportAllocs()
	b.SetBytes(int64(len(body)))

	for b.Loop() {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// readAllForBench drains r.Body fully. Errors are ignored in benchmarks.
func readAllForBench(r *http.Request) (int64, error) {
	buf := make([]byte, 4096)

	var total int64

	for {
		n, err := r.Body.Read(buf)
		total += int64(n)
		if err != nil {
			return total, nil
		}
	}
}
