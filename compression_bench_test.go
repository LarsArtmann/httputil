package httputil

import (
	"net/http"
	"strings"
	"testing"
)

func FuzzCompression(f *testing.F) {
	f.Add([]byte("hello world"), "gzip")
	f.Add([]byte(strings.Repeat("a", 1024)), "gzip, deflate")
	f.Add([]byte(""), "")

	cfg := DefaultCompressionConfig()
	inner := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte("response body"))
	})

	f.Fuzz(func(t *testing.T, body []byte, acceptEncoding string) {
		handler := Compression(cfg)(inner)

		req := newTestRequest(http.MethodGet, "/", "")
		req.Header.Set(headerAcceptEncoding, acceptEncoding)

		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusOK)
	})
}

func BenchmarkCompression(b *testing.B) {
	cfg := DefaultCompressionConfig()
	middleware := Compression(cfg)

	body := []byte(strings.Repeat("a", defaultCompressionMinSize*2))

	inner := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write(body)
	})

	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")
	req.Header.Set(headerAcceptEncoding, encodingGzip)

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
