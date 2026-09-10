package httputil

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"

	etag "github.com/larsartmann/go-etag/server"
	servertiming "github.com/larsartmann/httputil/server_timing"
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
	handler := Compression(cfg)(newWriteStatusHandler("hello world"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("Content-Encoding"))

	// Output: gzip
}

func ExampleDecompression() {
	var compressed bytes.Buffer

	zw := gzip.NewWriter(&compressed)

	_, _ = zw.Write([]byte("hello decompression"))
	_ = zw.Close()

	cfg := DefaultDecompressionConfig()
	handler := Decompression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		fmt.Println(string(body))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Output: hello decompression
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
	handler := servertiming.ServerTimingMiddleware()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stop := servertiming.MeasureServerTiming(r.Context(), "db")
			stop()
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get(servertiming.HeaderServerTiming) != "")

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

func ExampleETag() {
	handler := etag.New(etag.DefaultETagConfig())(newWriteStatusHandler("hello world"))

	// First request: the middleware computes and sets the ETag header.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	generated := rec.Header().Get("ETag")
	fmt.Println(generated != "")

	// Second request with matching If-None-Match: 304 Not Modified, empty body.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("If-None-Match", generated)
	handler.ServeHTTP(rec2, req2)
	fmt.Println(rec2.Code)

	// Output:
	// true
	// 304
}

func ExampleMaxBodySize() {
	handler := MaxBodySize(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader([]byte("this is way too long")),
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output: 413
}

func ExampleNonce() {
	handler := Nonce(
		DefaultNonceConfig(),
	)(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			// Use in templates: <script {{ NonceAttr }}>...</script>
			fmt.Println(NonceAttr(r) != "")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("Content-Security-Policy") != "")

	// Output:
	// true
	// true
}

// exampleMetricsRecorder captures the most recent observation for assertion
// in ExampleMetrics.
type exampleMetricsRecorder struct {
	lastMethod string
	lastPath   string
	lastStatus int
}

func (r *exampleMetricsRecorder) Record(method, path string, status int, _ time.Duration) {
	r.lastMethod = method
	r.lastPath = path
	r.lastStatus = status
}

func ExampleMetrics() {
	recorder := &exampleMetricsRecorder{}

	handler := Metrics(MetricsConfig{Recorder: recorder})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))

	fmt.Println(rec.Code, recorder.lastMethod, recorder.lastPath, recorder.lastStatus)

	// Output: 418 GET /hello 418
}

func ExampleHealthHandler() {
	rec := httptest.NewRecorder()
	HealthHandler()(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get("Content-Type"))
	fmt.Print(rec.Body.String())

	// Output:
	// 200
	// application/json
	// {"status":"up"}
}

func ExampleServer() {
	srv, err := NewServer(ServerConfig{
		Addr:            "127.0.0.1:0",
		ShutdownTimeout: time.Second,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello")
	}))
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	srv.Start()

	addr, ok := func() (string, bool) {
		deadline := time.Now().Add(time.Second)

		for time.Now().Before(deadline) {
			if listenAddr, listening := srv.ListenerAddr(); listening {
				return listenAddr.String(), true
			}

			time.Sleep(time.Millisecond)
		}

		return "", false
	}()
	if !ok {
		fmt.Println("server did not start")

		return
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://"+addr+"/",
		nil,
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(resp.StatusCode, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Println("error:", err)
	}

	// Output: 200 hello
}

func ExampleMiddlewareStack() {
	stack := NewMiddlewareStack()

	if err := stack.Add(MiddlewareRequestID, RequestID(DefaultRequestIDConfig())); err != nil {
		fmt.Println("error:", err)

		return
	}

	if err := stack.Add(
		MiddlewareSecurityHeaders,
		SecurityHeaders(DefaultSecurityHeadersConfig()),
	); err != nil {
		fmt.Println("error:", err)

		return
	}

	handler := stack.Build(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code, rec.Body.String(), stack.Names())

	// Output: 200 ok [request-id security-headers]
}
