package httputil

import (
	"compress/gzip"
	"io"
	"net/http"
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
func assertUncompressedResponse(t *testing.T, rec *ResponseRecorder, wantBody string) {
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
func assertGzipResponseBody(t *testing.T, rec *ResponseRecorder, wantBody string) {
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
func TestCompression_AbsentAcceptEncoding_FirstConfiguredCompressesWithHighestPriority(t *testing.T) {
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
