package httputil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func ExampleClientIP() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 70.41.3.18")
	req.RemoteAddr = "10.0.0.1:1234"

	fmt.Println(ClientIP(req))

	// Output: 203.0.113.1
}

func ExampleCORS() {
	cfg := DefaultCORSConfig()
	handler := CORS(cfg)(newNoOpHandler())

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output: 204
}

func ExampleChain() {
	wrapper := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Order", name)
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := newNoOpHandler()

	chain := Chain(handler, wrapper("first"), wrapper("second"))
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Header().Values("X-Order"))

	// Output: [first second]
}

func ExampleNewResponseRecorder() {
	inner := httptest.NewRecorder()
	rec := NewResponseRecorder(inner)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Status())

	// Output: 404
}
