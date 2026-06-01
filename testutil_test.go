package httputil

import (
	"bytes"
	"context"
	"log/slog"
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

// newTestLogger returns a slog.Logger that writes to a discard buffer.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
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

// assertSliceEqual checks that two string slices are element-wise equal.
func assertSliceEqual(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for i, v := range want {
		if got[i] != v {
			t.Errorf("got[%d] = %q, want %q", i, got[i], v)
		}
	}
}
