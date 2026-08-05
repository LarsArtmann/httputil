package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func FuzzETagConditional(f *testing.F) {
	f.Add("GET", "/", "\"abc123\"", "If-Match", "\"abc123\"")
	f.Add("GET", "/", "\"abc123\"", "If-None-Match", "\"abc123\"")
	f.Add("GET", "/", "\"abc123\"", "If-Match", "*")
	f.Add("GET", "/", "\"abc123\"", "If-None-Match", "W/\"weak\"")
	f.Add("GET", "/", "*", "If-Match", "\"different\"")

	f.Fuzz(func(t *testing.T, method, path, etagValue, headerName, headerValue string) {
		t.Parallel()

		if !isValidMethod(method) || !isValidPath(path) || !isValidHTTPToken(headerName) {
			t.Skip()
		}

		_ = etagValue

		cfg := DefaultETagConfig()
		handler := ETag(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
		}))

		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(headerName, headerValue)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code < 100 || rec.Code >= 600 {
			t.Errorf("invalid status code: %d", rec.Code)
		}
	})
}

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

func isValidMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions,
		http.MethodTrace:
		return true
	default:
		return false
	}
}

func isValidPath(path string) bool {
	if !strings.HasPrefix(path, "/") {
		return false
	}

	if strings.Contains(path, "%") {
		return false
	}

	for _, c := range path {
		if c <= 0x20 || c >= 0x7f {
			return false
		}
	}

	return true
}
