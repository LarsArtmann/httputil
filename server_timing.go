package httputil

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Server-Timing header (W3C Server Timing API).
// Spec: https://w3c.github.io/server-timing/
//
// Wire format (metrics joined by ", "):
//
//	Server-Timing: db;desc="Database";dur=53, cache;desc="Cache";dur=2, total;dur=120
//
// dur is measured in milliseconds (fractional allowed). The header MUST be set
// before the response body is committed, which is why the middleware wraps the
// http.ResponseWriter and injects it at the first WriteHeader/Write call.

// HeaderServerTiming is the HTTP header name for the W3C Server Timing API.
const HeaderServerTiming = "Server-Timing"

// ServerTiming collects named timing metrics and renders them as a
// Server-Timing response header value (W3C Server Timing API).
//
// It is concurrency-safe. Create one via [NewServerTiming] (or via
// [ServerTimingMiddleware], which stores it in the request context) and access
// it in handlers with [ServerTimingFromContext].
//
// All methods are nil-safe: a nil *ServerTiming (the value returned by
// ServerTimingFromContext when no middleware is active) makes every method a
// no-op, so handlers can call Record/Measure unconditionally without
// per-request nil checks.
type ServerTiming struct {
	mu      sync.Mutex
	metrics []serverTimingMetric
}

type serverTimingMetric struct {
	name string
	desc string
	dur  time.Duration // zero duration = omit dur parameter
}

// Enabled reports whether the collector is active.
// Returns false for a nil receiver — the natural "off" state.
func (st *ServerTiming) Enabled() bool { return st != nil }

// NewServerTiming creates an active collector.
func NewServerTiming() *ServerTiming {
	return &ServerTiming{}
}

// Record adds a named timing metric.
//
// name must be a valid HTTP token (RFC 7230); invalid characters are replaced
// with '_' so the emitted header is always well-formed. desc is an optional
// human-readable description (pass "" to omit it); it is escaped and quoted
// automatically, so it may contain commas/semicolons/quotes. A zero dur omits
// the dur parameter for that metric.
//
// Nil-safe: a no-op when st is nil (no middleware active).
// Safe to call concurrently from any goroutine.
func (st *ServerTiming) Record(name, desc string, dur time.Duration) {
	if st == nil {
		return
	}

	cleaned := sanitizeMetricName(name)
	if cleaned == "" {
		return
	}

	st.mu.Lock()
	st.metrics = append(st.metrics, serverTimingMetric{name: cleaned, desc: desc, dur: dur})
	st.mu.Unlock()
}

// Measure returns a function that records the elapsed time since it was
// called, under the given metric name. It is designed for the defer idiom:
//
//	defer st.Measure("db")()
//
// Nil-safe: returns a no-op function when st is nil.
//
// NOTE: the measured region must END before the response is committed
// (Write/WriteHeader) for the metric to appear in the header. See
// MeasureServerTiming for details.
func (st *ServerTiming) Measure(name string) func() {
	if st == nil {
		return func() {}
	}

	start := time.Now()

	return func() { st.Record(name, "", time.Since(start)) }
}

// MeasureWithDesc is like Measure but also records a description.
func (st *ServerTiming) MeasureWithDesc(name, desc string) func() {
	if st == nil {
		return func() {}
	}

	start := time.Now()

	return func() { st.Record(name, desc, time.Since(start)) }
}

// HeaderValue renders the collected metrics as a Server-Timing header value.
// Returns "" when nil or empty.
func (st *ServerTiming) HeaderValue() string {
	if st == nil {
		return ""
	}

	st.mu.Lock()
	metrics := st.metrics
	st.mu.Unlock()

	if len(metrics) == 0 {
		return ""
	}

	var b strings.Builder

	for i, m := range metrics {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(m.name)

		if m.desc != "" {
			b.WriteString(`;desc="`)
			b.WriteString(escapeQuotedString(m.desc))
			b.WriteByte('"')
		}

		if m.dur != 0 {
			b.WriteString(";dur=")
			b.WriteString(formatMillis(m.dur))
		}
	}

	return b.String()
}

// String is an alias for HeaderValue.
func (st *ServerTiming) String() string { return st.HeaderValue() }

// formatMillis renders a duration as milliseconds with the shortest
// round-tripping representation (e.g. 53ms -> "53", 0.5ms -> "0.5").
// The Server-Timing spec expresses dur in milliseconds; fractional values are
// permitted, so sub-millisecond timings are not lost.
func formatMillis(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)

	return strconv.FormatFloat(ms, 'f', -1, 64)
}

// escapeQuotedString escapes a value for an RFC 7230 quoted-string: backslash
// and double-quote are backslash-escaped. CR and LF are replaced with spaces
// to prevent HTTP header injection. All other bytes (including commas,
// semicolons, and spaces) are safe inside a quoted-string.
func escapeQuotedString(s string) string {
	if !strings.ContainsAny(s, `"\`+"\r\n") {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 4) //nolint:mnd // 4 = delimiter + quote overhead in Server-Timing format

	for i := range s {
		c := s[i]
		switch c {
		case '\\', '"':
			b.WriteByte('\\')
		case '\r', '\n':
			c = ' '
		}

		b.WriteByte(c)
	}

	return b.String()
}

// sanitizeMetricName enforces the RFC 7230 "token" rule for a Server-Timing
// metric name. Invalid bytes are replaced with '_'. Empty results are dropped.
func sanitizeMetricName(name string) string {
	const tchar = "!#$%&'*+-.^_`|~0123456789" +
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	if name == "" {
		return ""
	}

	cleaned := strings.Map(func(r rune) rune {
		if r < 128 && strings.ContainsRune(tchar, r) {
			return r
		}

		return '_'
	}, name)

	return cleaned
}

// ---------------------------------------------------------------------------
// Context integration
// ---------------------------------------------------------------------------

type serverTimingKey struct{}

// WithServerTiming stores a *ServerTiming in the context.
func WithServerTiming(ctx context.Context, st *ServerTiming) context.Context {
	return context.WithValue(ctx, serverTimingKey{}, st)
}

// ServerTimingFromContext retrieves the *ServerTiming stored by
// ServerTimingMiddleware (or WithServerTiming). Returns nil when no
// Server-Timing collector is present — all methods on a nil *ServerTiming are
// no-ops, so callers can use it directly without nil checks.
func ServerTimingFromContext(ctx context.Context) *ServerTiming {
	st, _ := ctx.Value(serverTimingKey{}).(*ServerTiming)

	return st
}

// RecordServerTiming is a context-aware shortcut for Record. Nil-safe.
func RecordServerTiming(ctx context.Context, name, desc string, dur time.Duration) {
	ServerTimingFromContext(ctx).Record(name, desc, dur)
}

// MeasureServerTiming is a context-aware shortcut for Measure. Nil-safe.
//
// Use it to time a region that completes BEFORE the response is written:
//
//	stop := httputil.MeasureServerTiming(r.Context(), "db")
//	result, err := db.Query(ctx)
//	stop()
//	renderResult(w, result) // response committed here — header includes db
//
// AVOID the defer idiom for non-streaming handlers:
//
//	defer httputil.MeasureServerTiming(r.Context(), "render")()
//	// ... writes response during this function ...
//	// ^ defer fires at return — AFTER the write — so "render" misses the header.
//
// The defer idiom DOES work for streaming responses (SSE) where the header is
// set once at connection start and the metric is recorded during the stream.
func MeasureServerTiming(ctx context.Context, name string) func() {
	return ServerTimingFromContext(ctx).Measure(name)
}

// ---------------------------------------------------------------------------
// ResponseWriter wrapper
// ---------------------------------------------------------------------------

// delegatingWriter wraps http.ResponseWriter and delegates Flush, Hijack,
// Push, and Unwrap to the underlying writer when available. Embed this in
// custom ResponseWriter wrappers to preserve SSE, WebSocket, and HTTP/2
// capabilities without duplicating the delegation boilerplate.
type delegatingWriter struct {
	http.ResponseWriter
}

// Flush delegates to the underlying Flusher, if available.
func (w delegatingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack delegates to the underlying Hijacker so WebSocket upgrades work
// through wrappers. Returns http.ErrNotSupported when unavailable.
func (w delegatingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}

	return h.Hijack() //nolint:wrapcheck // delegate to underlying Hijacker
}

// Push delegates HTTP/2 server push to the underlying Pusher, if available.
func (w delegatingWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts) //nolint:wrapcheck // delegate to underlying Pusher
	}

	return http.ErrNotSupported
}

// Unwrap exposes the underlying ResponseWriter so http.ResponseController
// (Go 1.20+) can locate Flusher/Hijacker/Pusher through wrapper chains.
func (w delegatingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// serverTimingWriter wraps http.ResponseWriter to inject the Server-Timing
// header at the moment the response is committed (first WriteHeader or Write).
// It delegates Flush/Hijack/Push so SSE, WebSocket, and HTTP/2 push continue
// to work transparently through the wrapper.
type serverTimingWriter struct {
	delegatingWriter

	st       *ServerTiming
	start    time.Time
	injected bool
	wrote    bool
}

func (w *serverTimingWriter) WriteHeader(code int) {
	w.flushHeader()
	w.wrote = true
	w.delegatingWriter.WriteHeader(code)
}

func (w *serverTimingWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.flushHeader()
		w.wrote = true
	}

	return w.delegatingWriter.Write(b) //nolint:wrapcheck // delegate to underlying ResponseWriter
}

// flushHeader finalizes the total metric and writes the Server-Timing header
// exactly once. Idempotent.
func (w *serverTimingWriter) flushHeader() {
	if w.injected {
		return
	}

	w.injected = true
	w.st.prependTotal("total", "Total request", time.Since(w.start))

	if h := w.st.HeaderValue(); h != "" {
		w.delegatingWriter.Header().Set(HeaderServerTiming, h)
	}
}

// Flush, Hijack, Push, and Unwrap are promoted from delegatingWriter.

// prependTotal inserts the total metric at the front of the collector. It is
// only called once, at flushHeader time, so the total reflects time-to-first
// byte (TTFB) — the standard semantics for a Server-Timing total.
func (st *ServerTiming) prependTotal(name, desc string, dur time.Duration) {
	if st == nil {
		return
	}

	st.mu.Lock()
	st.metrics = append(
		[]serverTimingMetric{{name: name, desc: desc, dur: dur}},
		st.metrics...,
	)
	st.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Wrap helper (used by middleware and by external integrators)
// ---------------------------------------------------------------------------

// WrapServerTiming wraps the ResponseWriter with Server-Timing injection and
// stores a *ServerTiming in the request context. Returns the wrapped writer
// and modified request. This is the building block used by
// [ServerTimingMiddleware].
func WrapServerTiming(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	st := NewServerTiming()
	ctx := WithServerTiming(r.Context(), st)
	wrapped := &serverTimingWriter{
		delegatingWriter: delegatingWriter{ResponseWriter: w},
		st:               st,
		start:            time.Now(),
	}

	return wrapped, r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// ServerTimingMiddleware enables the Server-Timing response header for every
// request.
//
// It measures the total request time-to-first-byte and exposes a *ServerTiming
// collector in the request context so downstream handlers can record
// sub-metrics:
//
//	mux.Use(httputil.ServerTimingMiddleware())
//	// …in a handler:
//	stop := httputil.MeasureServerTiming(r.Context(), "db")
//	db.Query(...)
//	stop()
//
// Server-Timing can leak internal performance details; gate it for
// debug/admin requests with [ServerTimingMiddlewareWhen].
func ServerTimingMiddleware() func(http.Handler) http.Handler {
	return ServerTimingMiddlewareWhen(func(*http.Request) bool { return true })
}

// ServerTimingMiddlewareWhen enables Server-Timing only for requests where pred
// returns true. When pred returns false (or is nil), the request is passed
// through with zero overhead: no ResponseWriter wrapping and no collector in
// context. Handlers calling Record/Measure are natural no-ops (nil-receiver),
// so no per-handler branching is needed.
//
// Use this to gate Server-Timing behind a debug flag, an admin role, or a
// request query/header check:
//
//	httputil.ServerTimingMiddlewareWhen(func(r *http.Request) bool {
//	    return r.URL.Query().Has("debug")
//	})
func ServerTimingMiddlewareWhen(pred func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if pred == nil || !pred(r) {
				// Zero-overhead passthrough: no collector in context, no
				// writer wrapping. nil-receiver methods are no-ops.
				next.ServeHTTP(w, r)

				return
			}

			wrapped, r2 := WrapServerTiming(w, r)
			next.ServeHTTP(wrapped, r2)
		})
	}
}
