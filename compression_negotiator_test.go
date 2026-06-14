package httputil

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestNegotiator_PrefersGzip verifies gzip wins by default order.
func TestNegotiator_PrefersGzip(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(DefaultWriterFactories())

	encoding, _, ok := neg.negotiateEncoding("gzip, deflate")
	if !ok {
		t.Fatal("negotiation failed")
	}

	if encoding != encodingGzip {
		t.Errorf("encoding = %q, want %q", encoding, encodingGzip)
	}
}

// TestNegotiator_RespectsQValue verifies q-values determine priority.
func TestNegotiator_RespectsQValue(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(DefaultWriterFactories())

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

	neg := buildNegotiator(DefaultWriterFactories())

	encoding, _, ok := neg.negotiateEncoding("gzip;q=0, deflate")
	if !ok {
		t.Fatal("negotiation failed")
	}

	if encoding != encodingDeflate {
		t.Errorf("encoding = %q, want %q (gzip should be disabled by q=0)", encoding, encodingDeflate)
	}
}

// TestNegotiator_EmptyHeader verifies empty header picks first configured
// (preferred encoding per server priority list, not alphabetical).
func TestNegotiator_EmptyHeader(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(DefaultWriterFactories())

	encoding, _, ok := neg.negotiateEncoding("")
	if !ok {
		t.Fatal("negotiation failed on empty header")
	}

	// gzip ranks higher than deflate in preferredEncodingOrder.
	if encoding != encodingGzip {
		t.Errorf("encoding = %q, want %q", encoding, encodingGzip)
	}
}

// TestNegotiator_UnsupportedEncoding verifies unsupported encodings are skipped.
func TestNegotiator_UnsupportedEncoding(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(DefaultWriterFactories())

	encoding, _, ok := neg.negotiateEncoding("br, deflate, snappy")
	if !ok {
		t.Fatal("negotiation failed")
	}

	if encoding != encodingDeflate {
		t.Errorf("encoding = %q, want %q (br and snappy are unsupported)", encoding, encodingDeflate)
	}
}

// TestNegotiator_AllExcludedFallsBackToIdentity verifies graceful fallback.
func TestNegotiator_AllExcludedFallsBackToIdentity(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(DefaultWriterFactories())

	encoding, _, ok := neg.negotiateEncoding("gzip;q=0, deflate;q=0")
	if !ok {
		t.Fatal("negotiation failed")
	}

	if encoding != encodingIdentity {
		t.Errorf("encoding = %q, want %q (identity fallback)", encoding, encodingIdentity)
	}
}

// TestNegotiator_WhitespaceHandled verifies header whitespace is tolerated.
func TestNegotiator_WhitespaceHandled(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(DefaultWriterFactories())

	encoding, _, ok := neg.negotiateEncoding("  gzip  ;  q=0.5  ,  deflate  ;  q=0.8  ")
	if !ok {
		t.Fatal("negotiation failed")
	}

	if encoding != encodingDeflate {
		t.Errorf("encoding = %q, want %q (whitespace should be trimmed)", encoding, encodingDeflate)
	}
}

// TestCompression_Deflate verifies deflate encoding works end-to-end.
func TestCompression_Deflate(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

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
	handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", defaultCompressionMinSize+1)))
	}))

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
	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty (passthrough)", got)
	}

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
