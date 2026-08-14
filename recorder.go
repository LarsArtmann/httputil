package httputil

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Middleware wraps an http.Handler to intercept or modify request flow.
type Middleware = func(http.Handler) http.Handler

// validateConfig logs a configuration validation error at startup. Called by
// all middleware constructors to surface misconfigurations early — the pattern
// is validate-and-log, not validate-and-abort, matching the CSRF and Nonce
// precedent: invalid configs produce a warning but still construct a working
// middleware (falling back to defaults where applicable).
//
// When err is classified (implements errorfamily.Coded and Classified), the
// machine-readable code, family, and domain are logged as structured fields
// so log consumers can route by them without parsing messages.
func validateConfig(configName string, err error) {
	if err == nil {
		return
	}

	attrs := []slog.Attr{slog.String("error", err.Error())}

	if coded, ok := errors.AsType[errorfamily.Coded](err); ok {
		attrs = append(attrs, slog.String("code", coded.ErrorCode()))

		if domain, ok := DomainOf(err); ok {
			attrs = append(attrs, slog.String("domain", string(domain)))
		}
	}

	if classified, ok := errors.AsType[errorfamily.Classified](err); ok {
		attrs = append(attrs, slog.String("family", classified.ErrorFamily().String()))
	}

	record := slog.NewRecord(time.Time{}, slog.LevelError, "httputil: "+configName+" validation failed", 0)
	record.AddAttrs(attrs...)

	_ = slog.Default().Handler().Handle(context.Background(), record)
}

// writeCommittedBody writes body bytes to a ResponseWriter whose status is
// already committed. Write failures at this point cannot be surfaced to the
// client or to the caller, so they are intentionally discarded: honest
// silence, not a swallowed bug. Callers must have called WriteHeader first.
func writeCommittedBody(w http.ResponseWriter, body []byte) {
	// Status already committed; write failure is unreportable ("honest silence").
	_, _ = w.Write(body)
}

// ResponseRecorder wraps an http.ResponseWriter to capture the status code.
// It also supports http.Flusher and http.Hijacker when the underlying
// ResponseWriter implements them.
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

// HeaderSnapshot returns a shallow copy of the response headers at the time
// of the call. Useful for inspecting headers after a handler has written them.
func (r *ResponseRecorder) HeaderSnapshot() http.Header {
	snapshot := make(http.Header, len(r.Header()))

	for key, values := range r.Header() {
		//nolint:makezero // pre-allocated for copy(), not append
		copied := make([]string, len(values))
		copy(copied, values)
		snapshot[key] = copied
	}

	return snapshot
}

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

	written, err := r.ResponseWriter.Write(b)
	if err != nil {
		return written, codeWriteFailed.WrapTransient(err, "response writer write failed").
			WithContext("status", strconv.Itoa(r.status))
	}

	return written, nil
}

// Flush delegates to the underlying ResponseWriter if it implements
// http.Flusher.
func (r *ResponseRecorder) Flush() {
	flushDelegate(r.ResponseWriter)
}

// Hijack delegates to the underlying ResponseWriter if it implements
// http.Hijacker.
func (r *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hijackDelegate(r.ResponseWriter)
}

// Chain wraps a handler with multiple middleware, applying them in reverse
// order so the first middleware in the variadic list is the outermost.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for _, mw := range slices.Backward(middlewares) {
		handler = mw(handler)
	}

	return handler
}
