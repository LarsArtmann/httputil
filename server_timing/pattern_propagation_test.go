package servertiming

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPatternPropagationThroughServerTimingMiddleware(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusOK)
	})

	var captured string
	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(resp, req)
			captured = req.Pattern
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	outer(ServerTimingMiddleware()(mux)).ServeHTTP(rec, req)

	if captured != "GET /users/{id}" {
		t.Errorf(
			"outer pattern = %q, want %q (pattern lost across ServerTiming fork)",
			captured, "GET /users/{id}",
		)
	}
}
