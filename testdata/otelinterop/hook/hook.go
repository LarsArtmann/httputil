// Package hook is the "Exception events without an SDK dependency" snippet from
// docs/integrations/otel.md: plain otelhttp plus httputil.RecoveryWithConfig
// bridges panics onto the active span. Keep it in sync with the guide.
package hook

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/larsartmann/httputil"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// New returns the recovery middleware with the documented OnPanic bridge.
func New(logger *slog.Logger) httputil.Middleware {
	return httputil.RecoveryWithConfig(httputil.RecoveryConfig{
		Logger: logger,
		OnPanic: func(r *http.Request, recovered any) {
			span := trace.SpanFromContext(r.Context())
			if err, ok := recovered.(error); ok {
				span.RecordError(err)
			} else {
				span.RecordError(fmt.Errorf("%v", recovered))
			}

			span.SetStatus(codes.Error, "panic")
		},
	})
}
