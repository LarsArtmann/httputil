package httputil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkHTTPRequestConstruction profiles the request-construction helpers
// used across the test suite, quantifying the cost behind the noctx linter's
// httptest.NewRequest warnings (source: `05-45:f27`, `00-51:f36`).
// httptest.NewRequest wraps http.ReadRequest on a synthetic wire request, so
// it is meaningfully heavier than http.NewRequestWithContext; the numbers
// document whether hoisting construction out of benchmark loops is worth it.
// Results (2026-08-30, Go 1.26.7 linux/amd64): recorded in
// docs/research/2026-08-30_httptest-newrequest-profiling.md.
func BenchmarkHTTPRequestConstruction(b *testing.B) {
	ctx := context.Background()

	b.Run("httptestNewRequest", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = httptest.NewRequest(http.MethodGet, "/", nil)
		}
	})

	b.Run("httptestNewRequestWithContext", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_ = httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		}
	})

	b.Run("httpNewRequestWithContext", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_, _ = http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		}
	})
}
