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

func assertQValue(t *testing.T, input string, want float64) {
	t.Helper()

	got, err := parseQValue(input)
	if err != nil {
		t.Errorf("parseQValue(%q) error = %v, want nil", input, err)

		return
	}

	diff := got - want
	if diff < -0.001 || diff > 0.001 {
		t.Errorf("parseQValue(%q) = %f, want %f", input, got, want)
	}
}

func assertQValueError(t *testing.T, input string) {
	t.Helper()

	_, err := parseQValue(input)
	if err == nil {
		t.Errorf("parseQValue(%q) error = nil, want error", input)
	}
}

// TestParseQValue_One verifies q-value "1" parses correctly.
func TestParseQValue_One(t *testing.T) {
	t.Parallel()

	assertQValue(t, "1", 1.0)
}

// TestParseQValue_Zero verifies q-value "0" parses correctly.
func TestParseQValue_Zero(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0", 0.0)
}

// TestParseQValue_Half verifies q-value "0.5" parses correctly.
func TestParseQValue_Half(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.5", 0.5)
}

// TestParseQValue_NineTenths verifies q-value "0.9" parses correctly.
func TestParseQValue_NineTenths(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.9", 0.9)
}

// TestParseQValue_OnePointZero verifies q-value "1.0" parses correctly.
func TestParseQValue_OnePointZero(t *testing.T) {
	t.Parallel()

	assertQValue(t, "1.0", 1.0)
}

// TestParseQValue_ThreeDecimals verifies q-value "0.001" parses correctly.
func TestParseQValue_ThreeDecimals(t *testing.T) {
	t.Parallel()

	assertQValue(t, "0.001", 0.001)
}

// TestParseQValue_EmptyError verifies empty q-value returns an error.
func TestParseQValue_EmptyError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "")
}

// TestParseQValue_TwoError verifies q-value "2" returns an error.
func TestParseQValue_TwoError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "2")
}

// TestParseQValue_OnePointFiveError verifies q-value "1.5" returns an error.
func TestParseQValue_OnePointFiveError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "1.5")
}

// TestParseQValue_AlphabeticError verifies q-value "abc" returns an error.
func TestParseQValue_AlphabeticError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "abc")
}

// TestParseQValue_DoubleDecimalError verifies q-value "0.5.5" returns an error.
func TestParseQValue_DoubleDecimalError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "0.5.5")
}

// TestParseQValue_TrailingCharsError verifies q-value "1.0a" returns an error.
func TestParseQValue_TrailingCharsError(t *testing.T) {
	t.Parallel()

	assertQValueError(t, "1.0a")
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

// TestGzipWriterFactory_ReturnsUsableWriter verifies the factory creates a working writer.
func TestGzipWriterFactory_ReturnsUsableWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder

	writer, err := GzipWriterFactory(gzip.DefaultCompression)(&buf)
	if err != nil {
		t.Fatalf("GzipWriterFactory() error = %v", err)
	}

	_, err = writer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := gzip.NewReader(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("gzip.NewReader error = %v", err)
	}
	defer func() { _ = reader.Close() }()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if string(decoded) != "hello world" {
		t.Errorf("decoded = %q, want %q", decoded, "hello world")
	}
}

// TestDeflateWriterFactory_ReturnsUsableWriter verifies the factory creates a working writer.
func TestDeflateWriterFactory_ReturnsUsableWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder

	writer, err := DeflateWriterFactory(gzip.DefaultCompression)(&buf)
	if err != nil {
		t.Fatalf("DeflateWriterFactory() error = %v", err)
	}

	_, err = writer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader := flate.NewReader(strings.NewReader(buf.String()))
	defer func() { _ = reader.Close() }()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if string(decoded) != "hello world" {
		t.Errorf("decoded = %q, want %q", decoded, "hello world")
	}
}
