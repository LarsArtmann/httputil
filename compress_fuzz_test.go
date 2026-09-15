package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func FuzzCompressWriterState(f *testing.F) {
	f.Add("gzip", "hello world", "text/plain")
	f.Add("deflate", "hello world", "text/plain")
	f.Add("identity", "hello world", "text/plain")
	f.Add("gzip", "", "text/plain")
	f.Add("gzip", strings.Repeat("a", 1000), "application/json")
	f.Add("br", "hello brotli", "text/plain")
	f.Add("GZIP; q=0.5, deflate", "hello wire", "text/plain")

	// Round-trip invariant (the repo rule for response transformers): the
	// response body must decode — via the negotiated Content-Encoding — to
	// exactly the bytes the handler wrote, or pass through byte-exact when
	// no encoding was negotiated. Decoders are bounded so a hostile stream
	// cannot balloon the runner.
	f.Fuzz(func(t *testing.T, encoding, body, contentType string) {
		t.Parallel()

		cfg := DefaultCompressionConfig()
		handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", encoding)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		gotEncoding := rec.Header().Get("Content-Encoding")
		gotBody := rec.Body.Bytes()

		switch gotEncoding {
		case "gzip":
			decoded, err := decodeBoundedGzip(gotBody, len(body)+1024)
			if err != nil {
				t.Errorf("gzip round-trip failed for body %q: %v", body, err)

				return
			}

			if !bytes.Equal(decoded, []byte(body)) {
				t.Errorf("gzip round-trip mismatch: got %q, want %q", decoded, body)
			}
		case "deflate":
			decoded, err := decodeBoundedFlate(gotBody, len(body)+1024)
			if err != nil {
				t.Errorf("deflate round-trip failed for body %q: %v", body, err)

				return
			}

			if !bytes.Equal(decoded, []byte(body)) {
				t.Errorf("deflate round-trip mismatch: got %q, want %q", decoded, body)
			}
		case "", "identity":
			if !bytes.Equal(gotBody, []byte(body)) {
				t.Errorf(
					"uncompressed response must be byte-exact: got %q, want %q (encoding %q)",
					gotBody, body, encoding,
				)
			}
		default:
			t.Errorf(
				"negotiated unexpected Content-Encoding %q from Accept-Encoding %q",
				gotEncoding,
				encoding,
			)
		}
	})
}

// decodeBoundedGzip gunzips at most limit bytes of r.
func decodeBoundedGzip(r []byte, limit int) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(r))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}

	defer func() { _ = gzipReader.Close() }()

	decoded, err := io.ReadAll(io.LimitReader(gzipReader, int64(limit)))
	if err != nil {
		return nil, fmt.Errorf("gzip read: %w", err)
	}

	return decoded, nil
}

// decodeBoundedFlate inflates at most limit bytes of raw-deflate data.
func decodeBoundedFlate(r []byte, limit int) ([]byte, error) {
	decoded, err := io.ReadAll(io.LimitReader(flate.NewReader(bytes.NewReader(r)), int64(limit)))
	if err != nil {
		return nil, fmt.Errorf("flate read: %w", err)
	}

	return decoded, nil
}

// FuzzNegotiatorWireFormat fuzzes raw Accept-Encoding header strings through
// the negotiator, complementing the q-value property tests with arbitrary
// wire input. Invariants: negotiation with the default factories (identity
// registered) always succeeds — a client that excludes every encoding falls
// back to identity, never to failure — and any accepted result is a
// registered canonical encoding name with a q-value in (0, 1].
func FuzzNegotiatorWireFormat(f *testing.F) {
	f.Add("gzip")
	f.Add("gzip, deflate, br")
	f.Add("gzip;q=0.5, deflate;q=0.8")
	f.Add("*")
	f.Add("gzip;q=0, deflate;q=0, identity;q=0")
	f.Add(" GZIP ; Q=0.5 ,")
	f.Add("x-gzip;q=abc, gzip;q=1.000")
	f.Add("gzip;q=1.001")
	f.Add("gzip;q=")
	f.Add(",,,")

	neg := newTestNegotiator()

	f.Fuzz(func(t *testing.T, header string) {
		encoding, quality, ok := neg.negotiateEncoding(header)

		if !ok {
			t.Errorf(
				"negotiation failed for header %q with identity registered, want identity fallback",
				header,
			)

			return
		}

		if _, registered := neg.factories[encoding]; !registered {
			t.Errorf("negotiated encoding %q is not registered (header %q)", encoding, header)
		}

		if quality <= 0 || quality > 1 {
			t.Errorf("negotiated q = %v outside (0,1] for header %q", quality, header)
		}

		// Selection micro-oracle: when the fuzzed header happens to name
		// exactly one registered encoding (case/whitespace variants), the
		// result must be that encoding — this pins actual selection, not
		// just internal consistency, so a negotiator that ignores the wire
		// input fails here. Trimming is HTTP OWS (SP/HTAB) only: control
		// characters like \v are not whitespace on the wire, and a header
		// such as "\vGZIP" is correctly an unsupported token (identity
		// fallback), which strings.TrimSpace would wrongly normalize to
		// "gzip" (found by this fuzz target, 2026-09-15).
		for name := range neg.factories {
			if strings.EqualFold(strings.Trim(header, " \t"), name) {
				if encoding != name {
					t.Errorf(
						"single-token header %q negotiated to %q, want %q",
						header, encoding, name,
					)
				}

				break
			}
		}
	})
}
