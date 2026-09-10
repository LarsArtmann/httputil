package httputil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BenchmarkMaxBodySize measures the middleware wrapping plus a full
// request-body read under the limit. The request is rebuilt every iteration
// because the body reader is consumed by the handler.
func BenchmarkMaxBodySize(b *testing.B) {
	middleware := MaxBodySize(1 << 20)

	body := []byte(strings.Repeat("a", 4096))

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
	}))

	b.ReportAllocs()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
