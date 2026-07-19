package httputil

import (
	"net/http"
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
