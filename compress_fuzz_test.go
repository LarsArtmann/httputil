package httputil

import (
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

	f.Fuzz(func(t *testing.T, encoding, body, contentType string) {
		t.Parallel()

		acceptEncoding := encoding
		if encoding != "gzip" && encoding != "deflate" && encoding != "identity" {
			acceptEncoding = "gzip"
		}

		cfg := DefaultCompressionConfig()
		handler := Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", acceptEncoding)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code < 100 || rec.Code >= 600 {
			t.Errorf("invalid status code: %d", rec.Code)
		}
	})
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
	})
}
