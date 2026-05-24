package httputil

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"slices"
)

// ResponseRecorder wraps an http.ResponseWriter to capture the status code.
// It also supports http.Flusher, http.Hijacker, and http.Pusher when the
// underlying ResponseWriter implements them.
type ResponseRecorder struct {
	http.ResponseWriter

	status int
	wrote  bool
}

// NewResponseRecorder creates a ResponseRecorder that defaults to 200 OK.
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{ResponseWriter: w, status: 0, wrote: false}
}

// Status returns the captured HTTP status code, or 0 if WriteHeader has not
// been called.
func (r *ResponseRecorder) Status() int { return r.status }

// WroteHeader reports whether WriteHeader has been called.
func (r *ResponseRecorder) WroteHeader() bool { return r.wrote }

// WriteHeader captures the status code and delegates to the underlying
// ResponseWriter.
func (r *ResponseRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}

	r.ResponseWriter.WriteHeader(code)
}

// Write implicitly sets status 200 if WriteHeader has not yet been called.
func (r *ResponseRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.status = http.StatusOK
		r.wrote = true
	}

	n, err := r.ResponseWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("response writer write: %w", err)
	}

	return n, nil
}

// Flush delegates to the underlying ResponseWriter if it implements
// http.Flusher.
func (r *ResponseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack delegates to the underlying ResponseWriter if it implements
// http.Hijacker.
func (r *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}

	conn, rw, err := h.Hijack()
	if err != nil {
		return conn, rw, fmt.Errorf("response writer hijack: %w", err)
	}

	return conn, rw, nil
}

// Push delegates to the underlying ResponseWriter if it implements
// http.Pusher.
func (r *ResponseRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return fmt.Errorf("push %q: %w", target, pusher.Push(target, opts))
	}

	return http.ErrNotSupported
}

// Chain wraps a handler with multiple middleware, applying them in reverse
// order so the first middleware in the variadic list is the outermost.
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for _, mw := range slices.Backward(middlewares) {
		handler = mw(handler)
	}

	return handler
}
