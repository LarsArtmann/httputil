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
