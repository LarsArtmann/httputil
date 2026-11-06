package endpoints

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newBenchRegistry(count int) *Registry {
	registry := NewRegistry()
	mux := http.NewServeMux()

	for i := range count {
		pattern := fmt.Sprintf("GET /resource-%d/{id}", i)

		if err := registry.Handle(mux, pattern, http.NotFoundHandler(), "Resource"); err != nil {
			panic(err)
		}
	}

	return registry
}

// BenchmarkRegistry_Routes measures a snapshot read: atomic load, copy, sort.
// The result is consumed so the call cannot be optimized away.
func BenchmarkRegistry_Routes(b *testing.B) {
	registry := newBenchRegistry(benchRouteCount)

	var sink int

	for b.Loop() {
		sink += len(registry.Routes())
	}

	if sink == 0 {
		b.Fatal("sink is zero; benchmark result was not consumed")
	}
}

// BenchmarkRegistry_Handler_JSON measures a full JSON listing request.
func BenchmarkRegistry_Handler_JSON(b *testing.B) {
	registry := newBenchRegistry(benchRouteCount)

	for b.Loop() {
		rec := httptest.NewRecorder()
		registry.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/endpoints", nil))

		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	}
}

// BenchmarkRegistry_Handler_HTML measures a full HTML listing request
// (negotiation plus per-field escaping).
func BenchmarkRegistry_Handler_HTML(b *testing.B) {
	registry := newBenchRegistry(benchRouteCount)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/endpoints", nil)
		req.Header.Set("Accept", "text/html")

		rec := httptest.NewRecorder()
		registry.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	}
}

// BenchmarkNegotiateRepresentation measures the Accept parsing alone.
func BenchmarkNegotiateRepresentation(b *testing.B) {
	browser := "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"

	for b.Loop() {
		_ = negotiateRepresentation(browser)
	}
}

// benchRouteCount is the registry size for the listing benchmarks.
const benchRouteCount = 50
