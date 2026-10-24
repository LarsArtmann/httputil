package httpspec

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// headerTraceparent is the canonical Go spelling of the W3C trace-context
// request header.
const headerTraceparent = "Traceparent"

// validTraceparent is a well-formed W3C trace-context header (version 00,
// 32-hex trace ID, 16-hex parent ID, sampled flag).
const validTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

// invalidTraceparent is deliberately outside the W3C grammar: receivers must
// treat it as if the header were absent.
const invalidTraceparent = "not-a-w3c-traceparent"

// headerTracestate is the canonical Go spelling of the W3C trace-context
// vendor-state header.
const headerTracestate = "Tracestate"

// validTracestate is a well-formed W3C tracestate (one vendor entry, the
// format tracing SDKs propagate across services).
const validTracestate = "vendor1=value1,vendor2=value2"

// invalidTracestate is deliberately outside the W3C grammar: receivers must
// treat it as if the header were absent.
const invalidTracestate = "=,;"

// patternProbeRoute is the ServeMux pattern propagation probes write into
// r.Pattern before the request enters the middleware chain.
const patternProbeRoute = "GET /httpspec/{probe}"

// probeStatus is deliberately unusual so pass-through checks cannot be
// satisfied by an accidental default status.
const probeStatus = http.StatusTeapot

// probeBody is the fixed body the probe handler writes.
const probeBody = "probe"

// probeRequestBody is the fixed body the request-invariance probe sends, so a
// middleware that consumes or rewrites the body is detected.
const probeRequestBody = `{"probe":true}`

// probeRoute is the request path the probes send.
const probeRoute = "/"

// Tracing spec names identify each handler-level tracing behavior check for
// use with [SkipSpec].
const (
	SpecNameTracingValidTraceparentAccepted  = "requests carrying a valid traceparent should not produce a server error"
	SpecNameTracingInvalidTraceparentIgnored = "a malformed traceparent should behave like an absent one"
	SpecNameTracingTracestateAccepted        = "requests carrying a valid tracestate should not produce a server error"
	SpecNameTracingInvalidTracestateIgnored  = "a malformed tracestate should behave like an absent one"
)

// TracingSpecs returns handler-level tracing behavior specs that can be
// composed into a spec run via [WithExtraSpecs]. They verify the contracts a
// handler owes a tracing layer: a valid W3C traceparent or tracestate never
// causes a server error, and a malformed one is ignored rather than rejected.
//
// These checks observe responses only, so they cannot see request-header
// passthrough or response-writer capabilities. Passthrough of both headers is
// verified at the middleware level by [RunTracing].
//
// The ignored-header checks compare two requests against the same handler, so
// run them against a stateless handler (no per-request counters or rate
// limiting).
func TracingSpecs() []Spec {
	return []Spec{
		{
			Name:     SpecNameTracingValidTraceparentAccepted,
			Category: CategoryHeaders,
			Check:    validTraceparentAcceptedCheck(),
		},
		{
			Name:     SpecNameTracingInvalidTraceparentIgnored,
			Category: CategoryHeaders,
			Check:    invalidTraceparentIgnoredCheck(),
		},
		{
			Name:     SpecNameTracingTracestateAccepted,
			Category: CategoryHeaders,
			Check:    validTracestateAcceptedCheck(),
		},
		{
			Name:     SpecNameTracingInvalidTracestateIgnored,
			Category: CategoryHeaders,
			Check:    invalidTracestateIgnoredCheck(),
		},
	}
}

func validTracestateAcceptedCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, probeRoute)
		req.Header.Set(headerTracestate, validTracestate)

		rec := serve(handler, req)

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"a request with a valid tracestate returned status %d; tracing headers must never cause a server error",
				rec.Code,
			)
		}

		return Pass()
	}
}

func invalidTracestateIgnoredCheck() Check {
	return func(handler http.Handler) Result {
		absent := serve(handler, mustRequest(http.MethodGet, probeRoute))

		req := mustRequest(http.MethodGet, probeRoute)
		req.Header.Set(headerTracestate, invalidTracestate)

		invalid := serve(handler, req)

		if invalid.Code != absent.Code {
			return Fail(
				"a malformed tracestate changed the status from %d to %d; W3C trace context says invalid headers are ignored",
				absent.Code,
				invalid.Code,
			)
		}

		if invalid.Body.String() != absent.Body.String() {
			return Fail(
				"a malformed tracestate changed the response body; W3C trace context says invalid headers are ignored",
			)
		}

		return Pass()
	}
}

func validTraceparentAcceptedCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, probeRoute)
		req.Header.Set(headerTraceparent, validTraceparent)

		rec := serve(handler, req)

		if rec.Code >= http.StatusInternalServerError {
			return Fail(
				"a request with a valid traceparent returned status %d; tracing headers must never cause a server error",
				rec.Code,
			)
		}

		return Pass()
	}
}

func invalidTraceparentIgnoredCheck() Check {
	return func(handler http.Handler) Result {
		absent := serve(handler, mustRequest(http.MethodGet, probeRoute))

		req := mustRequest(http.MethodGet, probeRoute)
		req.Header.Set(headerTraceparent, invalidTraceparent)

		invalid := serve(handler, req)

		if invalid.Code != absent.Code {
			return Fail(
				"a malformed traceparent changed the status from %d to %d; W3C trace context says invalid headers are ignored",
				absent.Code,
				invalid.Code,
			)
		}

		if invalid.Body.String() != absent.Body.String() {
			return Fail(
				"a malformed traceparent changed the response body; W3C trace context says invalid headers are ignored",
			)
		}

		return Pass()
	}
}

// RunTracing verifies a middleware chain against tracing contracts that a
// handler-level spec cannot observe. The middleware is applied to an internal
// probe handler in a fresh chain per check, so it can be the real tracing
// middleware or the full composed stack.
//
// Checks:
//   - a request's traceparent and tracestate reach the wrapped handler unchanged;
//   - the matched route pattern (r.Pattern) reaches the wrapped handler;
//   - an absent traceparent stays absent;
//   - the wrapped handler's method, path, and body arrive unmodified;
//   - the wrapped handler's status and body reach the client unchanged;
//   - http.Flusher, http.Pusher, and http.Hijacker survive the middleware's
//     writer wrapper.
func RunTracing(t *testing.T, middleware func(http.Handler) http.Handler) {
	t.Helper()

	if msg := tracingMiddlewareProblem(middleware); msg != "" {
		t.Fatal(msg)
	}

	t.Run("traceparent, tracestate, and route pattern reach the wrapped handler", func(t *testing.T) {
		t.Parallel()

		if msg := checkRequestPassthrough(middleware); msg != "" {
			t.Error(msg)
		}
	})

	t.Run("an absent traceparent stays absent", func(t *testing.T) {
		t.Parallel()

		if msg := checkAbsentTraceparent(middleware); msg != "" {
			t.Error(msg)
		}
	})

	t.Run("method, path, and body arrive unmodified", func(t *testing.T) {
		t.Parallel()

		if msg := checkRequestInvariance(middleware); msg != "" {
			t.Error(msg)
		}
	})

	t.Run("handler status and body pass through", func(t *testing.T) {
		t.Parallel()

		if msg := checkResponsePassthrough(middleware); msg != "" {
			t.Error(msg)
		}
	})

	t.Run("Flusher, Pusher, and Hijacker survive the wrapper", func(t *testing.T) {
		t.Parallel()

		if msg := checkWriterInterfaces(middleware); msg != "" {
			t.Error(msg)
		}
	})
}

// tracingMiddlewareProblem returns a failure message when the middleware handed
// to RunTracing cannot be checked.
func tracingMiddlewareProblem(middleware func(http.Handler) http.Handler) string {
	if middleware == nil {
		return "httpspec: RunTracing middleware must not be nil"
	}

	return ""
}

// checkRequestPassthrough returns a failure message, or "" when the check
// passes.
func checkRequestPassthrough(middleware func(http.Handler) http.Handler) string {
	probe := new(tracingProbe)

	req := mustRequest(http.MethodGet, probeRoute)
	req.Header.Set(headerTraceparent, validTraceparent)
	req.Header.Set(headerTracestate, validTracestate)
	req.Pattern = patternProbeRoute

	serve(middleware(probe.handler()), req)

	if probe.traceparent != validTraceparent {
		return fmt.Sprintf("traceparent at handler = %q, want %q", probe.traceparent, validTraceparent)
	}

	if probe.tracestate != validTracestate {
		return fmt.Sprintf("tracestate at handler = %q, want %q", probe.tracestate, validTracestate)
	}

	if probe.pattern != patternProbeRoute {
		return fmt.Sprintf(
			"r.Pattern at handler = %q, want %q (the middleware must propagate it)",
			probe.pattern,
			patternProbeRoute,
		)
	}

	return ""
}

// checkAbsentTraceparent returns a failure message, or "" when the check
// passes.
func checkAbsentTraceparent(middleware func(http.Handler) http.Handler) string {
	probe := new(tracingProbe)

	serve(middleware(probe.handler()), mustRequest(http.MethodGet, probeRoute))

	if probe.traceparent != "" {
		return fmt.Sprintf(
			"traceparent at handler = %q, want empty (the middleware must not invent one)",
			probe.traceparent,
		)
	}

	return ""
}

// checkResponsePassthrough returns a failure message, or "" when the check
// passes.
func checkResponsePassthrough(middleware func(http.Handler) http.Handler) string {
	probe := new(tracingProbe)
	rec := serve(middleware(probe.handler()), mustRequest(http.MethodGet, probeRoute))

	if rec.Code != probeStatus {
		return fmt.Sprintf("client status = %d, want %d", rec.Code, probeStatus)
	}

	if rec.Body.String() != probeBody {
		return fmt.Sprintf("client body = %q, want %q", rec.Body.String(), probeBody)
	}

	return ""
}

// checkWriterInterfaces returns a failure message, or "" when the check
// passes.
func checkWriterInterfaces(middleware func(http.Handler) http.Handler) string {
	probe := new(tracingProbe)
	outer := &interfaceProbeRecorder{ResponseRecorder: httptest.NewRecorder()}

	middleware(probe.handler()).ServeHTTP(outer, mustRequest(http.MethodGet, probeRoute))

	if !probe.canFlush {
		return "handler writer does not implement http.Flusher; streaming responses would break"
	}

	if !probe.canPush {
		return "handler writer does not implement http.Pusher; HTTP/2 server push would break"
	}

	if !probe.canHijack {
		return "handler writer does not implement http.Hijacker; WebSocket upgrades would break"
	}

	return ""
}

// checkRequestInvariance returns a failure message, or "" when the check
// passes. A tracing middleware must not rewrite the request it forwards.
func checkRequestInvariance(middleware func(http.Handler) http.Handler) string {
	probe := new(tracingProbe)

	req := mustRequest(http.MethodPost, probeRoute)
	req.Body = io.NopCloser(strings.NewReader(probeRequestBody))
	req.ContentLength = int64(len(probeRequestBody))

	serve(middleware(probe.handler()), req)

	if probe.method != http.MethodPost {
		return fmt.Sprintf("method at handler = %q, want %q", probe.method, http.MethodPost)
	}

	if probe.path != probeRoute {
		return fmt.Sprintf("path at handler = %q, want %q", probe.path, probeRoute)
	}

	if probe.body != probeRequestBody {
		return fmt.Sprintf(
			"body at handler = %q, want %q (a wrapper must not consume it)",
			probe.body,
			probeRequestBody,
		)
	}

	return ""
}

// tracingProbe records what a middleware chain delivers to its wrapped
// handler.
type tracingProbe struct {
	traceparent string
	tracestate  string
	pattern     string
	method      string
	path        string
	body        string
	canFlush    bool
	canPush     bool
	canHijack   bool
}

// handler returns the probe handler. It answers 418 with a fixed body so
// pass-through checks can assert an unusual status and body.
func (p *tracingProbe) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.traceparent = r.Header.Get(headerTraceparent)
		p.tracestate = r.Header.Get(headerTracestate)
		p.pattern = r.Pattern
		p.method = r.Method
		p.path = r.URL.Path
		p.body = readProbeBody(r)

		_, p.canFlush = w.(http.Flusher)
		_, p.canPush = w.(http.Pusher)
		_, p.canHijack = w.(http.Hijacker)

		w.WriteHeader(probeStatus)
		_, _ = w.Write([]byte(probeBody))
	})
}

// readProbeBody returns the request body, or "" when it is absent or
// unreadable. It reads at most one byte more than the probe body so a fuzzed or
// oversized request cannot exhaust memory.
func readProbeBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}

	sent, err := io.ReadAll(io.LimitReader(r.Body, int64(len(probeRequestBody)+1)))
	if err != nil {
		return ""
	}

	return string(sent)
}

// interfaceProbeRecorder is a ResponseRecorder that also implements
// http.Hijacker and http.Pusher, so writer-wrapper delegation can be observed
// without a real connection.
type interfaceProbeRecorder struct {
	*httptest.ResponseRecorder
}

// Hijack reports that hijacking is unsupported. The probes only type-assert
// the interface; they never call it.
func (interfaceProbeRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.ErrUnsupported
}

// Push reports that server push is unsupported. The probes only type-assert the
// interface; they never call it.
func (interfaceProbeRecorder) Push(string, *http.PushOptions) error {
	return errors.ErrUnsupported
}
