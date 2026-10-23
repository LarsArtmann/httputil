package httpspec

import (
	"net/http"
	"testing"
)

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
