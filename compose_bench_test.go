package httputil

import (
	"net/http"
	"testing"
)

// nopMiddleware returns next unchanged, the minimal composition unit.
func nopMiddleware(next http.Handler) http.Handler {
	return next
}

// BenchmarkCompose measures composition cost split by phase: Construct is the
// one-time Compose-plus-apply cost paid at wiring time; Serve is the
// steady-state request cost through an already-composed stack.
func BenchmarkCompose(b *testing.B) {
	handler := newWriteStatusHandler("")

	b.Run("Construct", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = Compose(nopMiddleware, nopMiddleware, nopMiddleware)(handler)
		}
	})

	b.Run("Serve", func(b *testing.B) {
		b.ReportAllocs()

		composed := Compose(nopMiddleware, nopMiddleware, nopMiddleware)(handler)
		req := newTestRequest(http.MethodGet, "/", "")

		for b.Loop() {
			rec := newRecorder()
			composed.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkMiddlewareStack_Middleware measures the cost of wrapping a
// MiddlewareStack as a single composable middleware: the per-apply copy of
// the stack's middleware slice plus the nested Chain construction.
func BenchmarkMiddlewareStack_Middleware(b *testing.B) {
	stack := NewMiddlewareStack()

	if err := stack.Add("nop-1", nopMiddleware); err != nil {
		b.Fatal(err)
	}

	if err := stack.Add("nop-2", nopMiddleware); err != nil {
		b.Fatal(err)
	}

	if err := stack.Add("nop-3", nopMiddleware); err != nil {
		b.Fatal(err)
	}

	handler := newWriteStatusHandler("")

	b.ReportAllocs()

	for b.Loop() {
		_ = stack.Middleware()(handler)
	}
}
