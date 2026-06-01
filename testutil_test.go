package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newNoOpHandler returns an http.HandlerFunc that does nothing.
func newNoOpHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

// newCountingHandler returns an http.HandlerFunc that sets called to true.
func newCountingHandler(called *bool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
	})
}

// newTestRequest creates an httptest.Request with context and Origin header set.
func newTestRequest(method, path, origin string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	return req
}

// newRecorder creates a new httptest.ResponseRecorder.
func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// newAppendingHandler returns an http.HandlerFunc that appends val to s.
func newAppendingHandler(s *[]string, val string) http.HandlerFunc {
	return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		*s = append(*s, val)
	})
}

// assertHeader checks that a response recorder has the expected header value.
func assertHeader(t *testing.T, rec *httptest.ResponseRecorder, key, want string) {
	t.Helper()

	if got := rec.Header().Get(key); got != want {
		t.Errorf("%s = %q, want %q", key, got, want)
	}
}
