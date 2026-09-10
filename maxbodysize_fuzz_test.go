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
// limits, matching http.MaxBytesReader semantics exactly: a body larger than
// max(maxBytes, 0) fails the read with http.MaxBytesError, every other body
// reads back byte-exact with no error. Zero and negative limits therefore
// reject every non-empty body while letting the empty body through (both
// boundaries probed against net/http).
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

		effectiveLimit := max(maxBytes, 0)

		if int64(len(body)) <= effectiveLimit {
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

		if _, ok := errors.AsType[*http.MaxBytesError](readErr); !ok {
			t.Errorf("read error = %v, want http.MaxBytesError", readErr)
		}
	})
}
