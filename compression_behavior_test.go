package httputil

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests specify the observable Accept-Encoding negotiation behavior mandated
// by RFC 7231. They describe what a client sees in the Content-Encoding response
// header, independent of the internal negotiator implementation.

// assertNegotiationForAcceptEncoding runs a Compression middleware with the
// default config against a request carrying the supplied Accept-Encoding
// value, then asserts the response status and Content-Encoding header. It is
// the single point through which q-value negotiation behavior is verified.
func assertNegotiationForAcceptEncoding(
	t *testing.T,
	acceptEncoding, wantEncoding string,
) {
	t.Helper()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, acceptEncoding)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, headerContentEncoding, wantEncoding)
}

// TestCompression_QValueZeroExcludesEncoding specifies that an encoding offered with
// q=0 is explicitly refused: a client sending "gzip;q=0, deflate" must not receive gzip.
func TestCompression_QValueZeroExcludesEncoding(t *testing.T) {
	t.Parallel()

	assertNegotiationForAcceptEncoding(t, "gzip;q=0, deflate", encodingDeflate)
}

// TestCompression_ServerPrefersGzipOverDeflateOnTie specifies the server-side priority
// tiebreak: when a client offers gzip and deflate with equal q-values, gzip wins
// (server order is brotli > zstd > gzip > deflate > identity).
func TestCompression_ServerPrefersGzipOverDeflateOnTie(t *testing.T) {
	t.Parallel()

	assertNegotiationForAcceptEncoding(t, "gzip, deflate", encodingGzip)
}

// TestCompression_AllQValuesZeroFallsBackToIdentity specifies that when the client
// excludes every compression encoding via q=0, the response is sent uncompressed.
func TestCompression_AllQValuesZeroFallsBackToIdentity(t *testing.T) {
	t.Parallel()

	assertNegotiationForAcceptEncoding(t, "gzip;q=0, deflate;q=0, identity", "")
}

// assertUncompressedResponse asserts the response carries no Content-Encoding
// header and the body is byte-identical to the plain text the handler wrote.
func assertUncompressedResponse(t *testing.T, rec *httptest.ResponseRecorder, wantBody string) {
	t.Helper()

	if got := rec.Header().Get(headerContentEncoding); got != "" {
		t.Errorf("Content-Encoding = %q, want none", got)
	}

	if got := rec.Body.String(); got != wantBody {
		t.Errorf("body was transformed: len = %d, want %d", len(got), len(wantBody))
	}
}

// assertGzipResponseBody asserts the response is gzip-encoded and gunzips to
// exactly wantBody.
func assertGzipResponseBody(t *testing.T, rec *httptest.ResponseRecorder, wantBody string) {
	t.Helper()

	assertHeader(t, rec, headerContentEncoding, encodingGzip)

	gzipReader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader error = %v", err)
	}

	defer func() { _ = gzipReader.Close() }()

	decoded, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if string(decoded) != wantBody {
		t.Errorf("gunzipped body = %d bytes, want the %d plain bytes", len(decoded), len(wantBody))
	}
}

// TestCompression_AbsentAcceptEncoding_ServesUncompressedByDefault specifies
// the default absent-header policy (issue #4): a request without an
// Accept-Encoding header receives the uncompressed representation, because
// minimal clients, probes, and intermediaries that do not declare support may
// not decompress.
func TestCompression_AbsentAcceptEncoding_ServesUncompressedByDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertUncompressedResponse(t, rec, strings.Repeat("a", defaultCompressionMinSize+1))
}

// TestCompression_EmptyAcceptEncodingValue_ServesUncompressedByDefault
// specifies RFC 7231 §5.3.4 for an empty Accept-Encoding value ("implies that
// the user agent does not want any content-coding in response"): the
// response is sent uncompressed.
func TestCompression_EmptyAcceptEncodingValue_ServesUncompressedByDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, "")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertUncompressedResponse(t, rec, strings.Repeat("a", defaultCompressionMinSize+1))
}

// TestCompression_AbsentAcceptEncoding_FirstConfiguredCompressesWithHighestPriority
// specifies the legacy opt-out: with AbsentEncodingFirstConfigured, a request
// without an Accept-Encoding header receives the highest-priority configured
// encoding (gzip for the default factories), fully decompressible to the
// original bytes.
func TestCompression_AbsentAcceptEncoding_FirstConfiguredCompressesWithHighestPriority(
	t *testing.T,
) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.AbsentEncoding = AbsentEncodingFirstConfigured
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertGzipResponseBody(t, rec, strings.Repeat("a", defaultCompressionMinSize+1))
}

// TestCompression_SniffsContentType_GzipPath_CTlessHandler pins the fix for
// the nosniff failure class: a handler that never sets Content-Type must
// still get net/http's sniffing answer on the COMPRESSED path, where the
// middleware commits the header before any body byte reaches net/http.
// Under X-Content-Type-Options: nosniff the missing type would make every
// browser render the page as plain text.
func TestCompression_SniffsContentType_GzipPath_CTlessHandler(t *testing.T) {
	t.Parallel()

	htmlBody := "<html><body>" + strings.Repeat("x", defaultCompressionMinSize) + "</body></html>"
	handler := Compression(DefaultCompressionConfig())(newWriteBodyHandler([]byte(htmlBody)))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertGzipResponseBody(t, rec, htmlBody)
	assertHeader(t, rec, headerContentType, "text/html; charset=utf-8")
}

// TestCompression_SniffsContentType_PlainClosePath_CTlessHandler covers the
// small-response path: a body below MinSize never starts compression and is
// drained in Close, which commits the header before writing the buffer — the
// sniff must run there too.
func TestCompression_SniffsContentType_PlainClosePath_CTlessHandler(t *testing.T) {
	t.Parallel()

	handler := Compression(DefaultCompressionConfig())(
		newWriteBodyHandler([]byte("<html><body>tiny</body></html>")),
	)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, headerContentEncoding, "")
	assertHeader(t, rec, headerContentType, "text/html; charset=utf-8")
}

// TestCompression_ExplicitContentType_NotOverridden pins the never-override
// contract: a handler-declared Content-Type is preserved byte-identically on
// the compressed path.
func TestCompression_ExplicitContentType_NotOverridden(t *testing.T) {
	t.Parallel()

	jsonBody := strings.Repeat(`{"k":"v"}`, defaultCompressionMinSize)
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentType, "application/problem+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonBody))
	})

	handler := Compression(DefaultCompressionConfig())(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertGzipResponseBody(t, rec, jsonBody)
	assertHeader(t, rec, headerContentType, "application/problem+json")
}

// TestCompression_SniffsContentType_BinaryStaysOctetStream pins that the
// sniff is honest for non-text bodies: undetectable bytes get
// application/octet-stream, not a fabricated text type.
func TestCompression_SniffsContentType_BinaryStaysOctetStream(t *testing.T) {
	t.Parallel()

	binaryBody := make([]byte, defaultCompressionMinSize+1)
	for i := range binaryBody {
		binaryBody[i] = byte(i % 256)
	}

	handler := Compression(DefaultCompressionConfig())(newWriteBodyHandler(binaryBody))

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, headerContentType, "application/octet-stream")
}

// TestCompression_NoContentTypeSniffed_WhenStatusForbidsBody mirrors
// net/http's bodyAllowedForStatus guard: 204 responses carry no body, so no
// Content-Type is invented for them even when the handler misbehaves and
// writes one.
func TestCompression_NoContentTypeSniffed_WhenStatusForbidsBody(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		_, _ = w.Write([]byte(strings.Repeat("x", defaultCompressionMinSize+1)))
	})

	handler := Compression(DefaultCompressionConfig())(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, headerContentType, "")
}

// TestCompression_NoContentTypeSniffed_WhenHandlerSetContentEncoding pins
// the issue-31753 guard: a handler that encoded its own body and set
// Content-Encoding manually must not have those encoded bytes sniffed on the
// plain path (the sniff would label the compressed octets, not the
// representation).
func TestCompression_NoContentTypeSniffed_WhenHandlerSetContentEncoding(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentEncoding, encodingGzip)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0x1f, 0x8b, 0x00, 0x01})
	})

	handler := Compression(DefaultCompressionConfig())(inner)

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, headerContentType, "")
}
