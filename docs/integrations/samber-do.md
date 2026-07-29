# Using httputil with samber/do

[**samber/do**](https://do.samber.dev/) is a dependency-injection container for Go built on generics. It handles service lifetimes, lazy construction, health checks, and graceful shutdown. It has no HTTP concerns at all — it never touches `http.Handler`, middleware, or the network. That is exactly the gap httputil fills.

Both libraries operate on plain Go types: `do` manages construction and lifecycle of your services at startup and shutdown; httputil manages per-request HTTP plumbing and server start/stop. Neither overlaps the other.

## Why they fit

- **No overlap:** `do` handles dependency wiring, lazy singletons, and shutdown orchestration. httputil handles compression, ETags, security headers, logging, recovery, CORS, rate limiting, and server lifecycle. Neither duplicates the other.
- **Clean boundary:** `do` runs in the composition root (startup and shutdown). httputil runs per-request (middleware chain). The two interact at exactly one point: graceful server shutdown.
- **Structural lifecycle match:** `httputil.Server.Shutdown(context.Context) error` satisfies `do.ShutdownerWithContextAndError` out of the box — no adapter, no wrapper, no `do` import in httputil. When the container shuts down, it discovers the HTTP server automatically and calls its `Shutdown` with **your** deadline.

## The lifecycle synergy

This is the one place the two libraries genuinely interlock. Every other integration point is just "build the handler inside a provider closure."

httputil's `Server` already has:

```go
func (srv *Server) Shutdown(ctx context.Context) error
```

samber/do's `ShutdownerWithContextAndError` interface is:

```go
type ShutdownerWithContextAndError interface {
    Shutdown(context.Context) error
}
```

These match exactly. When you invoke `*httputil.Server` from a `do` container, the container tracks it for shutdown. Later, `inj.ShutdownWithContext(ctx)` dispatches through a type switch that calls `srv.Shutdown(ctx)` — with your deadline, not a hardcoded one.

httputil does not import `do`, does not know about `do`, and never will (depguard forbids third-party dependencies). The match is a happy consequence of both libraries following the same Go convention for context-aware shutdown.

> [!IMPORTANT]
> Lazy services that were **never invoked** are **skipped** by `injector.Shutdown()`. Since you must invoke `*httputil.Server` to call `Start()`, the server is always tracked. Just make sure you invoke it — do not `Provide` it and then build it manually.

## Complete example

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	"github.com/larsartmann/httputil"
)

const defaultShutdownTimeout = 10 * time.Second

func main() {
	inj := do.New()

	// Eager foundation: the logger must exist before any service is built.
	do.ProvideValue(inj, slog.Default())

	// Lazy singletons: constructed on first Invoke, cached for the process lifetime.
	// MustInvoke inside provider closures is the canonical do pattern (build-time, not per-request).
	do.Provide(inj, NewUserRepo)
	do.Provide(inj, NewRouter)
	do.Provide(inj, NewHTTPServer)

	// Invoking the server transitively builds the router, repo, and logger.
	// The invoke also registers the server for automatic container shutdown.
	srv := do.MustInvoke[*httputil.Server](inj)

	errChan := srv.Start()

	// Block until a signal arrives or the server fails.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
	case err := <-errChan:
		slog.Error("server failed", "error", err)
	}

	// Graceful shutdown with a deadline.
	// do discovers *httputil.Server automatically (ShutdownerWithContextAndError)
	// and calls srv.Shutdown(ctx) for you — no manual call needed.
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if report := inj.ShutdownWithContext(ctx); !report.Succeed {
		for desc, shutdownErr := range report.Errors {
			slog.Error("shutdown failed",
				"scope", desc.ScopeName,
				"service", desc.Service,
				"error", shutdownErr,
			)
		}

		os.Exit(1)
	}
}

// NewUserRepo is a lazy singleton provider.
func NewUserRepo(i do.Injector) (*UserRepo, error) {
	return &UserRepo{}, nil
}

// NewRouter builds the stdlib ServeMux and registers routes.
// Dependencies are resolved inside the provider closure (canonical do pattern).
func NewRouter(i do.Injector) (*http.ServeMux, error) {
	repo := do.MustInvoke[*UserRepo](i)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, _ *http.Request) {
		users, _ := repo.All()
		_, _ = w.Write(users)
	})

	return mux, nil
}

// NewHTTPServer wraps the router with httputil middleware and returns a *httputil.Server.
// Recovery is first in Chain (outermost) so it catches panics from all other middleware.
func NewHTTPServer(i do.Injector) (*httputil.Server, error) {
	logger := do.MustInvoke[*slog.Logger](i)
	mux := do.MustInvoke[*http.ServeMux](i)

	wrapped := httputil.Chain(
		mux,
		httputil.Recovery(logger),
		httputil.Logging(logger),
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
		httputil.Compression(httputil.DefaultCompressionConfig()),
		httputil.ETag(httputil.DefaultETagConfig()),
		httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
	)

	return httputil.NewServer(httputil.DefaultServerConfig(), wrapped)
}

type UserRepo struct{}

func (r *UserRepo) All() ([]byte, error) {
	return []byte(`[{"id":1}]`), nil
}
```

> [!NOTE]
> This snippet lives in `docs/` rather than as an `example_do_test.go` because httputil's `depguard` lint policy restricts imports to the Go standard library, `go-error-family`, and `golang.org/x/time`. samber/do is a _user_ of httputil, not a dependency of it.

## Lifecycle flow

```
Startup:
  do.New()
    → ProvideValue(logger)                        eager foundation
    → Provide(NewUserRepo, NewRouter, NewHTTPServer)  lazy registrations
    → MustInvoke[*httputil.Server]                transitively builds all deps
    → srv.Start()                                 goroutine listens on :8080

Per-request:
  Client → httputil.Chain (Recovery → Logging → RequestID → Compression → ETag → SecurityHeaders)
         → *http.ServeMux → handler → UserRepo

Shutdown (SIGINT or server error):
  inj.ShutdownWithContext(ctx)
    → do discovers *httputil.Server via ShutdownerWithContextAndError
    → srv.Shutdown(ctx)                           graceful HTTP drain (your deadline)
    → other Shutdowner* services in reverse invocation order
```

httputil runs **outside** the router, so every route benefits from compression, ETags, security headers, logging, and recovery without any router-specific wiring. `do` runs **around** the entire application, owning construction and teardown.

## Combining with type-safe routing

Neither `do` nor httputil provides type-safe routing — by design. For typed handlers, input validation, path parameters, and OpenAPI generation, pair both with [huma](./huma.md). The three-layer split stays clean:

| Layer                   | Library   | Responsibility                                          |
| ----------------------- | --------- | ------------------------------------------------------- |
| Composition & lifecycle | samber/do | Dependency wiring, lazy singletons, graceful shutdown   |
| Type-safe routing       | huma      | Typed handlers, validation, OpenAPI from Go structs     |
| HTTP plumbing           | httputil  | Compression, ETags, security headers, logging, recovery |

`samber/do`'s generics (`[T any]`) cannot express heterogeneous handler signatures — Go has no variadic type parameters. Type-safe routing requires code generation (huma) or a generic builder, not a DI container. See the [research comparison](../research/2026-07-05_httputil-vs-huma.md) for details.

## See also

- [samber/do documentation](https://do.samber.dev/docs/getting-started)
- [samber/do API reference](https://pkg.go.dev/github.com/samber/do/v2)
- [Using httputil with huma](./huma.md)
- [httputil vs. huma — full comparison](../research/2026-07-05_httputil-vs-huma.md)
