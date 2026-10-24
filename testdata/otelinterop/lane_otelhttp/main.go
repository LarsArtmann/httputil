// Lane 2 of docs/integrations/otel.md — a bare otelhttp middleware composed
// with httputil. Keep this file in sync with the guide; the CI job
// `otel-interop` builds it so the published snippet cannot rot.
package main

import (
	"net/http"

	"github.com/larsartmann/httputil"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func newHandler(mux *http.ServeMux) http.Handler {
	return httputil.Compose(
		otelhttp.NewMiddleware("orders-api",
			// Only for internet-facing endpoints.
			otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return true }),
		),
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
	)(mux)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_ = http.ListenAndServe(":8080", newHandler(mux))
}
