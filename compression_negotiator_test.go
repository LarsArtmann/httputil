package httputil

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"testing"
)

// TestNegotiator_PrefersGzip verifies gzip wins by default order.
func TestNegotiator_PrefersGzip(t *testing.T) {
	t.Parallel()

	assertNegotiatedEncoding(
		t, newTestNegotiator(), "gzip, deflate", encodingGzip, "by default order",
	)
}

// TestNegotiator_RespectsQValue verifies q-values determine priority.
func TestNegotiator_RespectsQValue(t *testing.T) {
	t.Parallel()

	neg := newTestNegotiator()

	encoding, quality, ok := neg.negotiateEncoding("gzip;q=0.1, deflate;q=0.9")
	if !ok {
		t.Fatal("negotiation failed")
	}

	if encoding != encodingDeflate {
		t.Errorf("encoding = %q, want %q (higher q-value should win)", encoding, encodingDeflate)
	}

	if quality < 0.89 || quality > 0.91 {
		t.Errorf("q = %f, want ~0.9", quality)
	}
}

// TestNegotiator_QZeroDisables verifies q=0 excludes an encoding.
func TestNegotiator_QZeroDisables(t *testing.T) {
	t.Parallel()

	assertNegotiatedEncoding(
		t,
		newTestNegotiator(),
		"gzip;q=0, deflate",
		encodingDeflate,
		"gzip should be disabled by q=0",
	)
}

// TestNegotiator_EmptyHeader verifies empty header picks first configured
// (preferred encoding per server priority list, not alphabetical).
func TestNegotiator_EmptyHeader(t *testing.T) {
	t.Parallel()

	// gzip ranks higher than deflate in preferredEncodingOrder.
	assertNegotiatedEncoding(
		t, newTestNegotiator(), "", encodingGzip, "empty header picks top of priority order",
	)
}

// TestNegotiator_UnsupportedEncoding verifies unsupported encodings are skipped.
func TestNegotiator_UnsupportedEncoding(t *testing.T) {
	t.Parallel()

	assertNegotiatedEncoding(
		t,
		newTestNegotiator(),
		"br, deflate, snappy",
		encodingDeflate,
		"br and snappy are unsupported",
	)
}

// TestNegotiator_AllExcludedFallsBackToIdentity verifies graceful fallback.
func TestNegotiator_AllExcludedFallsBackToIdentity(t *testing.T) {
	t.Parallel()

	assertNegotiatedEncoding(
		t,
		newTestNegotiator(),
		"gzip;q=0, deflate;q=0",
		encodingIdentity,
		"identity fallback when all excluded",
	)
}

// TestNegotiator_WhitespaceHandled verifies header whitespace is tolerated.
func TestNegotiator_WhitespaceHandled(t *testing.T) {
	t.Parallel()

	assertNegotiatedEncoding(
		t,
		newTestNegotiator(),
		"  gzip  ;  q=0.5  ,  deflate  ;  q=0.8  ",
		encodingDeflate,
		"whitespace should be trimmed",
	)
}

// TestCompression_Deflate verifies deflate encoding works end-to-end.
func TestCompression_Deflate(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "deflate")

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	encoding := rec.Header().Get("Content-Encoding")
	if encoding != "deflate" {
		t.Fatalf("Content-Encoding = %q, want deflate", encoding)
	}

	flateReader := flate.NewReader(rec.Body)
	defer func() { _ = flateReader.Close() }()

	decoded, err := io.ReadAll(flateReader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if len(decoded) != defaultCompressionMinSize+1 {
		t.Errorf("decoded length = %d, want %d", len(decoded), defaultCompressionMinSize+1)
	}
}

// TestCompression_QValuePicksDeflate verifies q-value routing.
func TestCompression_QValuePicksDeflate(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "gzip;q=0.1, deflate;q=0.9")

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	encoding := rec.Header().Get("Content-Encoding")
	if encoding != "deflate" {
		t.Errorf("Content-Encoding = %q, want deflate (q=0.9 should win)", encoding)
	}
}

// TestCompression_CustomFactory verifies the WriterFactory plugin.
func TestCompression_CustomFactory(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{
		MinSize: 1,
		Level:   gzip.DefaultCompression,
		WriterFactories: map[string]WriterFactory{
			"identity": passthroughFactory,
		},
	}
	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set("Accept-Encoding", "identity")

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	// No compression should occur.
	assertHeader(t, rec, "Content-Encoding", "")

	if got := rec.Body.String(); got != "hello world" {
		t.Errorf("body = %q, want %q", got, "hello world")
	}
}

// TestCompressionConfig_Validate_EmptyFactories verifies error on no factories.
func TestCompressionConfig_Validate_EmptyFactories(t *testing.T) {
	t.Parallel()

	cfg := CompressionConfig{
		MinSize:         1,
		Level:           gzip.DefaultCompression,
		WriterFactories: map[string]WriterFactory{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty factories")
	}
}

// TestNegotiator_SingleTokenFastPath verifies a single clean encoding token
// (the common non-browser case) resolves directly without q-value parsing.
func TestNegotiator_SingleTokenFastPath(t *testing.T) {
	t.Parallel()

	assertNegotiatedEncoding(
		t, newTestNegotiator(), "gzip", encodingGzip, "single-token fast path",
	)
}

// TestNegotiator_EmptyOrderReturnsFalse covers the len(order)==0 branch of
// negotiateEmptyHeader. A negotiator with no configured encodings cannot
// serve anything.
func TestNegotiator_EmptyOrderReturnsFalse(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(map[string]WriterFactory{})

	_, _, ok := neg.negotiateEncoding("")
	if ok {
		t.Error("negotiateEncoding(empty) with no encodings = true, want false")
	}
}

// TestNegotiator_FallbackNoIdentity covers the branch of fallbackToIdentity
// where identity is not registered. When all encodings are q=0 and identity
// is absent from the factory map, negotiation fails.
func TestNegotiator_FallbackNoIdentity(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(map[string]WriterFactory{
		encodingGzip:    GzipWriterFactory(gzip.DefaultCompression),
		encodingDeflate: DeflateWriterFactory(gzip.DefaultCompression),
	})

	_, _, ok := neg.negotiateEncoding("gzip;q=0, deflate;q=0")
	if ok {
		t.Error("negotiateEncoding with all q=0 and no identity = true, want false")
	}
}

// TestNegotiator_ScanAcceptEncoding_OnlyUnsupported covers the scanAcceptEncoding
// path where the header contains only unsupported encodings (not q=0, just
// not in the factory map), causing bestName to remain empty.
func TestNegotiator_ScanAcceptEncoding_OnlyUnsupported(t *testing.T) {
	t.Parallel()

	neg := newTestNegotiator()

	_, _, ok := neg.negotiateEncoding("br, zstd, lz4")
	if ok {
		t.Error("negotiateEncoding with only unsupported encodings = true, want false")
	}
}

// BenchmarkNegotiateEncoding measures negotiation cost for the common
// single-token header (fast path) versus multi-token browser-style headers.
func BenchmarkNegotiateEncoding(b *testing.B) {
	neg := buildNegotiator(DefaultWriterFactories())

	cases := []struct {
		name   string
		header string
	}{
		{"single_token", "gzip"},
		{"multi_token", "gzip, deflate, br"},
		{"qvalues", "gzip;q=0.1, deflate;q=0.9"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			for range b.N {
				_, _, _ = neg.negotiateEncoding(tc.header)
			}
		})
	}
}
