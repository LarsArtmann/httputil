package httputil

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"
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
	wrapper := func(name string) Middleware {
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

func ExampleCompression() {
	cfg := CompressionConfig{MinSize: 1, Level: -2}
	handler := Compression(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("Content-Encoding"))

	// Output: gzip
}

func ExampleETag() {
	cfg := DefaultETagConfig()
	handler := ETag(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("ETag") != "")

	// Output: true
}

func ExampleRequestID() {
	cfg := DefaultRequestIDConfig()
	handler := RequestID(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		fmt.Println(id != "")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Output: true
}

func ExampleSecurityHeaders() {
	cfg := DefaultSecurityHeadersConfig()
	handler := SecurityHeaders(cfg)(newNoOpHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("X-Content-Type-Options"))

	// Output: nosniff
}

func ExampleRecovery() {
	logger := slog.New(slog.DiscardHandler)
	handler := Recovery(logger)(newPanicHandler("test panic"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output: 500
}

func ExampleTimeout() {
	handler := Timeout(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		fmt.Println(ok)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Output: true
}

func ExampleLogging() {
	logger := slog.New(slog.DiscardHandler)
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output: 200
}

func ExampleCSRFMiddleware() {
	handler := CSRFMiddleware(
		CSRFConfig{},
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output: 200
}

func ExampleServerTimingMiddleware() {
	handler := ServerTimingMiddleware()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stop := MeasureServerTiming(r.Context(), "db")
			stop()
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get(HeaderServerTiming) != "")

	// Output:
	// 200
	// true
}

func ExampleKeyedRateLimiterMiddleware() {
	cfg := KeyedRateLimiterConfig{
		Limit:        100,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
	}
	handler := KeyedRateLimiterMiddleware(
		cfg,
	)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output: 200
}
