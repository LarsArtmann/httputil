package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
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
