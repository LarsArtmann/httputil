package httputil

import (
	"net/http"
	"testing"
)

// These tests specify the observable Accept-Encoding negotiation behavior mandated
// by RFC 7231. They describe what a client sees in the Content-Encoding response
// header, independent of the internal negotiator implementation.

// TestCompression_QValueZeroExcludesEncoding specifies that an encoding offered with
// q=0 is explicitly refused: a client sending "gzip;q=0, deflate" must not receive gzip.
func TestCompression_QValueZeroExcludesEncoding(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, "gzip;q=0, deflate")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, headerContentEncoding, encodingDeflate)
}

// TestCompression_ServerPrefersGzipOverDeflateOnTie specifies the server-side priority
// tiebreak: when a client offers gzip and deflate with equal q-values, gzip wins
// (server order is brotli > zstd > gzip > deflate > identity).
func TestCompression_ServerPrefersGzipOverDeflateOnTie(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, "gzip, deflate")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, headerContentEncoding, encodingGzip)
}

// TestCompression_AllQValuesZeroFallsBackToIdentity specifies that when the client
// excludes every compression encoding via q=0, the response is sent uncompressed.
func TestCompression_AllQValuesZeroFallsBackToIdentity(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	handler := Compression(cfg)(newWriteLargeBodyHandler())

	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, "gzip;q=0, deflate;q=0, identity")

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, headerContentEncoding, "")
}
