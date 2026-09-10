package httputil

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzMaxBodySize pins the middleware contract for arbitrary bodies and
// limits, matching http.MaxBytesReader semantics exactly: a body of
// len(body) <= maxBytes bytes reads back byte-exact with no error, and a body
// larger than the limit fails the read with http.MaxBytesError. Zero and
// negative limits therefore reject every non-empty body (discovered by this
// fuzz target: a negative limit allows the empty body through, it does not
// error unconditionally).
func FuzzMaxBodySize(f *testing.F) {
	f.Add([]byte("hello"), int64(5))
	f.Add([]byte("hello"), int64(0))
	f.Add([]byte(""), int64(0))
	f.Add([]byte("exactly5"), int64(8))
	f.Add([]byte("beyond"), int64(-1))

	f.Fuzz(func(t *testing.T, body []byte, maxBytes int64) {
		var (
			got     []byte
			readErr error
		)

		handler := MaxBodySize(maxBytes)(http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			got, readErr = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if int64(len(body)) <= maxBytes {
			if readErr != nil {
				t.Errorf("body of %d bytes under limit %d failed: %v", len(body), maxBytes, readErr)

				return
			}

			if !bytes.Equal(got, body) {
				t.Errorf("read %d bytes, want byte-exact %d", len(got), len(body))
			}

			return
		}

		if readErr == nil {
			t.Errorf("body of %d bytes over limit %d read without error", len(body), maxBytes)

			return
		}

		var tooLarge *http.MaxBytesError
		if !errors.As(readErr, &tooLarge) {
			t.Errorf("read error = %v, want http.MaxBytesError", readErr)
		}
	})
}
