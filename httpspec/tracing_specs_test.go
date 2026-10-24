package httpspec

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"testing"
)

// passthroughWriter preserves http.Flusher and http.Hijacker the way httputil's
// own response wrappers do.
type passthroughWriter struct {
	http.ResponseWriter
}

// Flush delegates to the wrapped writer when it can flush.
func (p passthroughWriter) Flush() {
	if flusher, ok := p.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack delegates to the wrapped writer when it can hijack.
func (p passthroughWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := p.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}

	return nil, nil, errors.ErrUnsupported
}

// passthroughWriterMiddleware wraps the response writer in a type that keeps the
// optional interfaces reachable.
func passthroughWriterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(passthroughWriter{ResponseWriter: w}, r)
	})
}

// headerStampingMiddleware sets a response header before delegating, the way
// RequestID and CORS do.
func headerStampingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Probe-Route", "chain")
		next.ServeHTTP(w, r)
	})
}

// recoveringMiddleware converts a panic into a 500, the way Recovery does.
func recoveringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// identityMiddleware is the trivially correct middleware: it changes nothing.
func identityMiddleware(next http.Handler) http.Handler {
	return next
}

// plainWriter hides every optional interface (Flusher, Hijacker) behind the
// bare ResponseWriter contract.
type plainWriter struct {
	http.ResponseWriter
}

// traceparentStrippingMiddleware deletes the W3C header before delegating,
// simulating a wrapper that consumes trace context.
func traceparentStrippingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(headerTraceparent)
		next.ServeHTTP(w, r)
	})
}

// plainWriterMiddleware wraps the response writer in a type that implements
// only http.ResponseWriter.
func plainWriterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(plainWriter{ResponseWriter: w}, r)
	})
}

// flushOnlyWriter exposes http.Flusher while hiding http.Hijacker.
type flushOnlyWriter struct {
	http.ResponseWriter
}

// Flush delegates to the wrapped writer when it can flush.
func (f flushOnlyWriter) Flush() {
	if flusher, ok := f.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// flushOnlyMiddleware wraps the response writer so Flusher survives and
// Hijacker does not.
func flushOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(flushOnlyWriter{ResponseWriter: w}, r)
	})
}

// routePatternDroppingMiddleware clears the matched route pattern, simulating a
// wrapper that rebuilds the request.
func routePatternDroppingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Pattern = ""
		next.ServeHTTP(w, r)
	})
}

// traceparentInventingMiddleware fabricates a traceparent when the client sent
// none, simulating a wrapper that misreports trace context.
func traceparentInventingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerTraceparent) == "" {
			r.Header.Set(headerTraceparent, validTraceparent)
		}

		next.ServeHTTP(w, r)
	})
}

// bodyTamperingMiddleware lets the handler answer, then appends to the body.
func bodyTamperingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		_, _ = w.Write([]byte("tampered"))
	})
}

func TestTracingSpecs_PassWithPlainHandler(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	for _, spec := range TracingSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()

			result := spec.Check(handler)
			if !result.OK {
				t.Errorf("spec %q failed: %s", spec.Name, result.Message)
			}
		})
	}
}

func TestTracingSpecs_ValidTraceparent_FailsOnServerError(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerTraceparent) != "" {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	})

	result := validTraceparentAcceptedCheck()(handler)
	if result.OK {
		t.Error("spec passed for a handler that returns 500 on a valid traceparent, want failure")
	}
}

func TestTracingSpecs_InvalidTraceparent_FailsOnBodyChange(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerTraceparent) == invalidTraceparent {
			_, _ = w.Write([]byte("rejected"))

			return
		}

		_, _ = w.Write([]byte("ok"))
	})

	result := invalidTraceparentIgnoredCheck()(handler)
	if result.OK {
		t.Error("spec passed for a handler that reacts to a malformed traceparent, want failure")
	}
}

func TestRunTracing_PassesWithIdentityMiddleware(t *testing.T) {
	t.Parallel()

	RunTracing(t, identityMiddleware)
}

// TestRunTracing_DetectsStrippedTraceparent mutation-probes the check: the
// same check that passes for identity must fail when the header is consumed.
func TestRunTracing_DetectsStrippedTraceparent(t *testing.T) {
	t.Parallel()

	msg := checkRequestPassthrough(traceparentStrippingMiddleware)
	if msg == "" {
		t.Fatal("checkRequestPassthrough passed for a header-stripping middleware, want failure")
	}
}

// TestRunTracing_DetectsDroppedWriterInterfaces mutation-probes the check: a
// wrapper that hides Flusher/Hijacker must be reported.
func TestRunTracing_DetectsDroppedWriterInterfaces(t *testing.T) {
	t.Parallel()

	msg := checkWriterInterfaces(plainWriterMiddleware)
	if msg == "" {
		t.Fatal("checkWriterInterfaces passed for a writer that drops Flusher/Hijacker, want failure")
	}
}

func TestRunTracing_DetectsBrokenResponsePassthrough(t *testing.T) {
	t.Parallel()

	swallowingMiddleware := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}

	msg := checkResponsePassthrough(swallowingMiddleware)
	if msg == "" {
		t.Fatal("checkResponsePassthrough passed for a middleware that swallows the handler response, want failure")
	}
}

// TestRunTracing_DetectsDroppedRoutePattern mutation-probes the pattern half of
// the request-passthrough check.
func TestRunTracing_DetectsDroppedRoutePattern(t *testing.T) {
	t.Parallel()

	msg := checkRequestPassthrough(routePatternDroppingMiddleware)
	if msg == "" {
		t.Fatal("checkRequestPassthrough passed for a middleware that clears r.Pattern, want failure")
	}
}

// TestRunTracing_DetectsInventedTraceparent mutation-probes the absent-header
// invariance check.
func TestRunTracing_DetectsInventedTraceparent(t *testing.T) {
	t.Parallel()

	msg := checkAbsentTraceparent(traceparentInventingMiddleware)
	if msg == "" {
		t.Fatal("checkAbsentTraceparent passed for a middleware that invents a traceparent, want failure")
	}
}

// TestRunTracing_DetectsTamperedBody mutation-probes the body half of the
// response-passthrough check.
func TestRunTracing_DetectsTamperedBody(t *testing.T) {
	t.Parallel()

	msg := checkResponsePassthrough(bodyTamperingMiddleware)
	if msg == "" {
		t.Fatal("checkResponsePassthrough passed for a middleware that appends to the body, want failure")
	}
}

// TestRunTracing_DetectsDroppedHijacker mutation-probes the Hijacker half of
// the writer-interface check.
func TestRunTracing_DetectsDroppedHijacker(t *testing.T) {
	t.Parallel()

	msg := checkWriterInterfaces(flushOnlyMiddleware)
	if msg == "" {
		t.Fatal("checkWriterInterfaces passed for a writer that drops Hijacker, want failure")
	}
}

// TestRunTracing_RejectsNilMiddleware covers the guard that stops RunTracing
// from panicking on a nil chain.
func TestRunTracing_RejectsNilMiddleware(t *testing.T) {
	t.Parallel()

	if msg := tracingMiddlewareProblem(nil); msg == "" {
		t.Fatal("tracingMiddlewareProblem accepted a nil middleware, want a failure message")
	}
}

// TestTracingSpecs_InvalidTraceparent_FailsOnStatusChange mutation-probes the
// status half of the invalid-traceparent spec.
func TestTracingSpecs_InvalidTraceparent_FailsOnStatusChange(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerTraceparent) == invalidTraceparent {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	})

	result := invalidTraceparentIgnoredCheck()(handler)
	if result.OK {
		t.Error("spec passed for a handler that changes status on a malformed traceparent, want failure")
	}
}

// TestRunTracing_PassesOnComposedChain runs the contracts against a realistic
// stack: a header-stamping policy layer, a panic-to-500 layer, and a delegating
// writer wrapper, in the order the integration guide prescribes.
func TestRunTracing_PassesOnComposedChain(t *testing.T) {
	t.Parallel()

	chain := func(next http.Handler) http.Handler {
		return headerStampingMiddleware(recoveringMiddleware(passthroughWriterMiddleware(next)))
	}

	RunTracing(t, chain)
}

// TestRunTracing_ComposedChainDetectsRegression mutation-probes the composed
// chain: replacing the delegating wrapper with one that drops the optional
// interfaces must be reported even though the other layers are unchanged.
func TestRunTracing_ComposedChainDetectsRegression(t *testing.T) {
	t.Parallel()

	chain := func(next http.Handler) http.Handler {
		return headerStampingMiddleware(recoveringMiddleware(plainWriterMiddleware(next)))
	}

	msg := checkWriterInterfaces(chain)
	if msg == "" {
		t.Fatal("checkWriterInterfaces passed for a composed chain that drops Flusher/Hijacker, want failure")
	}
}
