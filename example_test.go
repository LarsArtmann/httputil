package httputil

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
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
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "api response")
	})
	handler := CORS(cfg)(apiHandler)

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

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

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
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})
	handler := Compression(cfg)(helloHandler)

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
	handler := SecurityHeaders(
		cfg,
	)(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("X-Content-Type-Options"))

	// Output: nosniff
}

func ExampleRecovery() {
	logger := slog.New(slog.DiscardHandler)
	handler := Recovery(logger)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))

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

func ExampleCSRFTokenFormField() {
	handler := CSRFMiddleware(
		CSRFConfig{},
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `<form method="post">`+CSRFTokenFormField(r)+`</form>`)
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	body := rec.Body.String()
	fmt.Println(
		strings.HasPrefix(
			body,
			`<form method="post"><input type="hidden" name="csrf_token" value="`,
		),
	)
	fmt.Println(strings.HasSuffix(body, `"></form>`))

	// Output:
	// true
	// true
}

func ExampleCSRFTokenHXHeaders() {
	handler := CSRFMiddleware(
		CSRFConfig{},
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "<body "+CSRFTokenHXHeaders(r)+">")
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(strings.HasPrefix(rec.Body.String(), `<body hx-headers='`))

	// Output: true
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

func Example_conditionalRequests() {
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})
	handler := etag.New(etag.DefaultETagConfig())(helloHandler)

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

func ExampleCompose() {
	// Composing an empty list yields the identity middleware: the handler
	// passes through unchanged.
	handler := Compose()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code, rec.Body.String())

	// Output: 200 ok
}

func ExampleCompose_ordered() {
	// Middleware apply in declaration order: the first entry becomes the
	// outermost wrapper.
	addHeader := func(name, value string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(name, value)
				next.ServeHTTP(w, r)
			})
		}
	}

	composed := Compose(
		addHeader("X-Outer", "outer"),
		addHeader("X-Inner", "inner"),
	)

	handler := composed(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(
		rec.Code,
		rec.Body.String(),
		rec.Header().Get("X-Outer"),
		rec.Header().Get("X-Inner"),
	)

	// Output: 200 ok outer inner
}

func ExampleMiddlewareFunc_Then() {
	// MiddlewareFunc lets a plain middleware function carry the Then method,
	// so middleware and handler chain fluently without a call to Chain.
	var addHeader MiddlewareFunc = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Trace", "abc")
			next.ServeHTTP(w, r)
		})
	}

	handler := addHeader.Then(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code, rec.Body.String(), rec.Header().Get("X-Trace"))

	// Output: 200 ok abc
}

func ExampleDomainOf() {
	cfg := DefaultCORSConfig()
	cfg.MaxAge = -1

	err := cfg.Validate()
	if err != nil {
		domain, ok := DomainOf(err)
		fmt.Println(domain, ok)
	}

	// Output: cors true
}

func ExampleInDomain() {
	cfg := DefaultCORSConfig()
	cfg.MaxAge = -1

	err := cfg.Validate()
	if err != nil && InDomain(err, Domain("cors")) {
		fmt.Println("fix the CORS configuration, then retry")
	}

	// Output: fix the CORS configuration, then retry
}

func ExampleChain_composition() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	// Chain applies middlewares in declaration order (first = outermost), so
	// Recovery outermost catches panics from everything inside it.
	handler := Chain(
		mux,
		Recovery(slog.New(slog.DiscardHandler)),
		RequestID(DefaultRequestIDConfig()),
		CORS(DefaultCORSConfig()),
	)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Origin", "https://example.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String() == "ok\n")
	fmt.Println(rec.Header().Get("X-Request-ID") != "")
	fmt.Println(rec.Header().Get("Access-Control-Allow-Origin"))

	// Output:
	// 200
	// true
	// true
	// *
}

func ExampleChain_nonceSecurityHeaders() {
	const staticCSP = "default-src 'self'"

	staticCfg := DefaultSecurityHeadersConfig()
	staticCfg.ContentSecurityPolicy = staticCSP

	nonceCfg := DefaultNonceConfig()
	nonceCfg.CSPBuilder = RecommendedCSPWithNonce

	// Nonce inner to SecurityHeaders: both write Content-Security-Policy on
	// the request path, so the nonce-bearing policy overwrites the static one.
	handler := Chain(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}),
		SecurityHeaders(staticCfg),
		Nonce(nonceCfg),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	csp := rec.Header().Get("Content-Security-Policy")
	fmt.Println(strings.Contains(csp, "'nonce-"))
	fmt.Println(csp == staticCSP)

	// Output:
	// true
	// false
}

func ExampleCORS_privateNetwork() {
	cfg := DefaultCORSConfig()
	cfg.AllowPrivateNetwork = true

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// A Chrome Local Network Access preflight: OPTIONS with the request
	// headers Chrome sends before fetching a more-private subresource. The
	// middleware-generated preflight answers 204 and carries the LNA header;
	// actual requests never do.
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://public.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Allow-Private-Network", "true")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get("Access-Control-Allow-Private-Network"))

	// Output:
	// 204
	// true
}

func ExampleKeyedRateLimiterMiddleware_maxKeys() {
	cfg := KeyedRateLimiterConfig{
		Limit:        1,
		Window:       time.Minute,
		KeyExtractor: KeyExtractorFromRemoteAddr(),
		MaxKeys:      1,
	}

	handler := KeyedRateLimiterMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := func(remoteAddr string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		return rec.Code
	}

	fmt.Println(request("10.0.0.1:1000"))
	fmt.Println(request("10.0.0.1:1000"))
	fmt.Println(request("10.0.0.2:1000"))
	fmt.Println(request("10.0.0.1:1000"))

	// Output:
	// 200
	// 429
	// 200
	// 200
}

func ExampleDecompression_maxSize() {
	var compressed bytes.Buffer

	zw := gzip.NewWriter(&compressed)

	_, _ = zw.Write([]byte("17 bytes payload"))
	_ = zw.Close()

	request := func(cfg DecompressionConfig, report func(n int, err error)) {
		handler := Decompression(cfg)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			report(len(body), err)
		}))

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressed.Bytes()))
		req.Header.Set("Content-Encoding", "gzip")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	// The MaxDecompressionSize zero value selects the 16 MiB default; there
	// is no unlimited option.
	request(DecompressionConfig{}, func(n int, err error) {
		fmt.Println(n, err == nil)
	})

	// A tiny limit trips bomb protection: the read fails once the
	// decompressed body exceeds it.
	request(DecompressionConfig{MaxDecompressionSize: 5}, func(_ int, err error) {
		fmt.Println(err != nil)
	})

	// Output:
	// 16 true
	// true
}

func ExampleErrCSRFInvalid() {
	// Context clones keep matching their sentinel — errors.Is matches by code
	// and family — so handlers can classify rejections precisely.
	rejection := ErrCSRFInvalid.WithContext("path", "/login")
	fmt.Println(errors.Is(rejection, ErrCSRFInvalid))

	// Plain wrapping composes with the same sentinel.
	err := fmt.Errorf("posting /login: %w", rejection)
	fmt.Println(errors.Is(err, ErrCSRFInvalid))

	// Output:
	// true
	// true
}

func ExampleInDomain_retryDecision() {
	cfg := DefaultCORSConfig()
	cfg.MaxAge = -1

	err := cfg.Validate()

	domain, ok := DomainOf(err)
	fmt.Println(ok, domain)

	// Rejection-family errors are never retryable: retrying the same input
	// cannot succeed, so fix the configuration instead.
	fmt.Println(errorfamily.Classify(err).IsRetryable())

	// Output:
	// true cors
	// false
}

func ExampleClientIP_trust() {
	// ClientIP reads X-Forwarded-For blindly. Here the header is
	// attacker-set and the real peer is 203.0.113.7; only trust the header
	// behind a reverse proxy that strips or overwrites it.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	req.RemoteAddr = "203.0.113.7:4444"

	fmt.Println(ClientIP(req))

	// Output: 9.9.9.9
}

func ExampleNewResponseRecorder_unwritten() {
	rec := NewResponseRecorder(httptest.NewRecorder())

	// Status reports 0 before any WriteHeader call; WroteHeader distinguishes
	// "no status set" from a real status of 0.
	fmt.Println(rec.Status(), rec.WroteHeader())

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Status(), rec.WroteHeader())

	// Output:
	// 0 false
	// 204 true
}

func ExampleCompression_absentEncoding() {
	build := func(policy AbsentEncodingPolicy) http.Handler {
		cfg := CompressionConfig{MinSize: 1, AbsentEncoding: policy}

		return Compression(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("hello world"))
		}))
	}

	// A request without Accept-Encoding, served by both policies.
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Default (AbsentEncodingIdentity): serve uncompressed.
	rec := httptest.NewRecorder()
	build(AbsentEncodingIdentity).ServeHTTP(rec, req)
	fmt.Printf("%q\n", rec.Header().Get("Content-Encoding"))

	// AbsentEncodingFirstConfigured: restore the pre-v1.2 behavior and pick
	// the highest-priority configured encoding anyway.
	rec2 := httptest.NewRecorder()
	build(AbsentEncodingFirstConfigured).ServeHTTP(rec2, req)
	fmt.Println(rec2.Header().Get("Content-Encoding"))

	// Output:
	// ""
	// gzip
}

func ExampleCSRFConfig_sameSiteFallback() {
	// SameSite=None without Secure produces a cookie current browsers refuse
	// to store (rfc6265bis §5.7), so cookie-writing paths fall back to
	// Secure=true while keeping the None intent. The deletion cookie here is
	// emitted as if configured with Secure: true.
	rec := httptest.NewRecorder()
	InvalidateCSRFCookie(rec, CSRFConfig{SameSite: http.SameSiteNoneMode, Secure: false})

	cookies := rec.Result().Cookies()

	fmt.Println(cookies[0].Name, cookies[0].Secure, cookies[0].SameSite == http.SameSiteNoneMode)

	// Output: csrf_token true true
}
