// Package otelinterop proves the claim docs/integrations/otel.md makes: the
// httpspec tracing contracts hold against the real OpenTelemetry middlewares,
// not only against the fakes used inside the httpspec module.
package otelinterop_test

import (
	"net/http"
	"testing"

	appkitotel "github.com/larsartmann/go-appkit/otel"
	"github.com/larsartmann/httputil"
	"github.com/larsartmann/httputil/httpspec"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestRunTracing_OtelhttpMiddleware(t *testing.T) {
	t.Parallel()

	httpspec.RunTracing(t, otelhttp.NewMiddleware("orders-api"))
}

func TestRunTracing_OtelhttpPublicEndpoint(t *testing.T) {
	t.Parallel()

	httpspec.RunTracing(t, otelhttp.NewMiddleware(
		"public-api",
		otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return true }),
	))
}

func TestRunTracing_AppkitMiddleware(t *testing.T) {
	t.Parallel()

	httpspec.RunTracing(t, appkitotel.Middleware())
}

// TestRunTracing_ComposedChain runs the contracts against the full ordering the
// guide prescribes: tracing outermost, Recovery inside the span, request policy
// below it.
func TestRunTracing_ComposedChain(t *testing.T) {
	t.Parallel()

	chain := httputil.Compose(
		otelhttp.NewMiddleware("orders-api"),
		httputil.RecoveryWithConfig(httputil.RecoveryConfig{
			Logger:  nil,
			OnPanic: func(*http.Request, any) {},
		}),
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
		httputil.CORS(httputil.DefaultCORSConfig()),
	)

	httpspec.RunTracing(t, chain)
}
