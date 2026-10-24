// Lane 1 of docs/integrations/otel.md — go-appkit/otel owns the SDK and the
// provider, httputil owns the middleware. Keep this file in sync with the guide;
// the CI job `otel-interop` builds it so the published snippet cannot rot.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	appkitotel "github.com/larsartmann/go-appkit/otel"
	"github.com/larsartmann/httputil"
)

func main() {
	logger := slog.Default()

	provider, err := appkitotel.Setup(
		appkitotel.WithService("orders-api", "1.0.0", os.Getenv("POD_NAME")),
		appkitotel.WithOTLP(),
	)
	if err != nil {
		logger.Error("otel setup", "error", err)
		os.Exit(1)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = provider.Shutdown(shutdownCtx)
	}()

	handler := httputil.Compose(
		appkitotel.Middleware(),
		appkitotel.Recovery(logger),
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
		httputil.CORS(httputil.DefaultCORSConfig()),
	)(newMux())

	_ = http.ListenAndServe(":8080", handler)
}

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
