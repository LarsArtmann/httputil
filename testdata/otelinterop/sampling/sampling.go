// Package sampling is the "Sampling" snippet from docs/integrations/otel.md:
// the provider owns the decision, and a ratio sampler must be wrapped in
// ParentBased so distributed traces stay continuous. Keep it in sync with the
// guide; the CI job `otel-interop` builds it.
package sampling

import (
	"context"
	"fmt"
	"os"

	"github.com/larsartmann/go-appkit/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Setup configures a provider that samples 10% of new traces and follows the
// parent decision for everything else.
func Setup(ctx context.Context) (func(context.Context) error, error) {
	provider, err := otel.Setup(
		otel.WithService("orders-api", "1.0.0", os.Getenv("POD_NAME")),
		otel.WithOTLP(),
		otel.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
	)
	if err != nil {
		return nil, fmt.Errorf("otel setup: %w", err)
	}

	return provider.Shutdown, nil
}
