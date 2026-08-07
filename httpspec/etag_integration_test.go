package httpspec

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/httputil"
)

// TestETagMiddleware_PassesHTTPSpec verifies that an ETag-wrapped handler
// passes the full standard httpspec suite and additional ETag-specific specs.
// This ensures the ETag middleware does not break any standard HTTP behavior.
func TestETagMiddleware_PassesHTTPSpec(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	})

	handler := httputil.ETag(httputil.DefaultETagConfig())(mux)

	Run(t, handler, WithExtraSpecs(
		Spec{
			Name:     "GET responses should include an ETag header",
			Category: CategoryHeaders,
			Check: func(h http.Handler) Result {
				rec := serveETagCheck(h, http.MethodGet, "")

				if rec.Header().Get("ETag") == "" {
					return Fail("GET / did not set an ETag header")
				}

				return Pass()
			},
		},
		Spec{
			Name:     "matching If-None-Match should return 304 Not Modified",
			Category: CategoryRouting,
			Check: func(h http.Handler) Result {
				rec := serveETagCheck(h, http.MethodGet, "")
				etag := rec.Header().Get("ETag")
				if etag == "" {
					return Fail("GET / did not set an ETag header for 304 test")
				}

				rec304 := serveETagCheck(h, http.MethodGet, etag)

				if rec304.Code != http.StatusNotModified {
					return Fail(
						"GET / with matching If-None-Match returned %d, want 304",
						rec304.Code,
					)
				}

				return Pass()
			},
		},
		Spec{
			Name:     "POST responses should not include an ETag header",
			Category: CategoryMethods,
			Check: func(h http.Handler) Result {
				rec := serveETagCheck(h, http.MethodPost, "")

				if rec.Header().Get("ETag") != "" {
					return Fail("POST / set an ETag header, want none")
				}

				return Pass()
			},
		},
	))
}

// serveETagCheck sends a request to handler with the given method and optional
// If-None-Match header value. Returns the response recorder.
func serveETagCheck(handler http.Handler, method, ifNoneMatch string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/", nil)

	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	return serve(handler, req)
}
