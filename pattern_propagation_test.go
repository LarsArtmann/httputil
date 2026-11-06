package httputil

import (
	"net/http"
	"testing"
	"time"
)

// patternCaptureMiddleware returns an outer middleware that records the route
// pattern visible on the request it handed down, after the wrapped handler has
// returned — the same read point otelhttp uses for span naming.
func patternCaptureMiddleware(captured *string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(resp, req)
			*captured = req.Pattern
		})
	}
}

// newPatternMux returns a mux with one pattern route and a handler that
// records the pattern as seen from inside the route.
func newPatternMux(innerPattern *string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(resp http.ResponseWriter, req *http.Request) {
		*innerPattern = req.Pattern
		resp.WriteHeader(http.StatusOK)
	})

	return mux
}

// assertPatternSurvivesFork serves one request through patternCaptureMiddleware
// wrapping mw wrapping the pattern mux, then asserts the route pattern is
// visible on both sides of the fork. name identifies the forking middleware
// in failure messages.
func assertPatternSurvivesFork(t *testing.T, name string, mw Middleware) {
	t.Helper()

	var outerPattern, innerPattern string
	handler := patternCaptureMiddleware(&outerPattern)(mw(newPatternMux(&innerPattern)))

	rec := newRecorder()
	handler.ServeHTTP(rec, newTestRequest(http.MethodGet, "/users/42", ""))

	if innerPattern != "GET /users/{id}" {
		t.Errorf("inner pattern = %q, want %q", innerPattern, "GET /users/{id}")
	}

	if outerPattern != "GET /users/{id}" {
		t.Errorf(
			"outer pattern = %q, want %q (pattern lost across %s fork)",
			outerPattern, "GET /users/{id}", name,
		)
	}
}

func TestRequestID_PropagatesRoutePattern(t *testing.T) {
	t.Parallel()

	assertPatternSurvivesFork(t, "RequestID", RequestID(DefaultRequestIDConfig()))
}

func TestTimeout_PropagatesRoutePattern(t *testing.T) {
	t.Parallel()

	assertPatternSurvivesFork(t, "Timeout", Timeout(time.Second))
}

func TestClientIP_PropagatesRoutePattern(t *testing.T) {
	t.Parallel()

	assertPatternSurvivesFork(t, "ClientIPMiddleware", ClientIPMiddleware)
}

func TestNonce_PropagatesRoutePattern(t *testing.T) {
	t.Parallel()

	assertPatternSurvivesFork(t, "Nonce", Nonce(DefaultNonceConfig()))
}

func TestPatternPropagationThroughFullForkChain(t *testing.T) {
	t.Parallel()

	var outerPattern, innerPattern string
	chain := Compose(
		patternCaptureMiddleware(&outerPattern),
		RequestID(DefaultRequestIDConfig()),
		Timeout(time.Second),
		ClientIPMiddleware,
		Nonce(DefaultNonceConfig()),
	)(newPatternMux(&innerPattern))

	rec := newRecorder()
	chain.ServeHTTP(rec, newTestRequest(http.MethodGet, "/users/42", ""))

	if innerPattern != "GET /users/{id}" {
		t.Errorf("inner pattern = %q, want %q", innerPattern, "GET /users/{id}")
	}

	if outerPattern != "GET /users/{id}" {
		t.Errorf(
			"outer pattern = %q, want %q (pattern lost across the four-fork chain)",
			outerPattern, "GET /users/{id}",
		)
	}
}

func TestPatternStaysEmptyWhenNoRouteMatches(t *testing.T) {
	t.Parallel()

	var outerPattern string
	chain := patternCaptureMiddleware(&outerPattern)(
		RequestID(DefaultRequestIDConfig())(http.NewServeMux()),
	)

	rec := newRecorder()
	chain.ServeHTTP(rec, newTestRequest(http.MethodGet, "/nomatch", ""))

	if outerPattern != "" {
		t.Errorf("outer pattern = %q, want empty for unmatched route", outerPattern)
	}
}
