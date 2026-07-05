# Using httputil with Huma

[**Huma**](https://huma.rocks/) is a declarative REST/RPC API framework for Go that generates OpenAPI 3.1 + JSON Schema from your types. It deliberately ships **no middleware** (v2 removed its v1 middleware package) — it expects you to bring your own for cross-cutting concerns. That is exactly httputil's job.

Both libraries target the **same Go 1.22+ `http.ServeMux`** via the [`humago`](https://pkg.go.dev/github.com/danielgtaylor/huma/v2/adapters/humago) adapter, so no third-party router sits in the critical path.

## Why they fit

- **Same foundation:** `humago.New` wraps `http.ServeMux`; httputil wraps `http.Handler`. Same router, same signature, zero glue.
- **No overlap:** huma handles typed handlers, input validation, and OpenAPI generation. httputil handles compression, ETags, security headers, logging, recovery, CORS, and server lifecycle. Neither duplicates the other.
- **Clean boundary:** httputil operates _outside_ the router (per-request plumbing); huma operates _inside_ the router (per-operation contracts). The layers never collide.

## Complete example

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/larsartmann/httputil"
)

// GreetingOutput is the response model. Huma generates JSON Schema from it.
type GreetingOutput struct {
	Body struct {
		Message string `json:"message" doc:"Greeting message"`
	}
}

func main() {
	// 1. Create the stdlib router — shared by huma and httputil.
	router := http.NewServeMux()

	// 2. Wrap it with Huma to get typed operations + OpenAPI generation.
	api := humago.New(router, huma.DefaultConfig("My API", "1.0.0"))

	huma.Get(api, "/greeting/{name}", func(ctx context.Context, input *struct {
		Name string `path:"name" maxLength:"30" doc:"Name to greet"`
	}) (*GreetingOutput, error) {
		resp := &GreetingOutput{}
		resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)

		return resp, nil
	})

	// 3. Wrap the router with httputil middleware (first = outermost).
	handler := httputil.Chain(
		router,
		httputil.Logging(slog.Default()),
		httputil.Recovery(slog.Default()),
		httputil.Compression(httputil.DefaultCompressionConfig()),
		httputil.ETag(httputil.DefaultETagConfig()),
		httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
		httputil.CORS(httputil.DefaultCORSConfig()),
	)

	// 4. Start the server.
	http.ListenAndServe(":8080", handler)
}
```

> [!NOTE]
> This snippet lives in `docs/` rather than as an `example_huma_test.go` because httputil's `depguard` lint policy restricts imports to the Go standard library and `go-error-family`. Huma is a _user_ of httputil, not a dependency of it.

## Request flow

```
Client → httputil.Chain (CORS → Security → ETag → Compression → Recovery → Logging)
       → http.ServeMux
       → humago adapter → huma operation handler (typed input → validation → OpenAPI)
```

httputil runs **outside** the router, so every huma operation benefits from compression, ETags, security headers, logging, and recovery without any huma-specific wiring.

## See also

- [httputil vs. huma — full comparison](../research/2026-07-05_httputil-vs-huma.md)
- [Huma docs](https://huma.rocks/)
- [humago adapter reference](https://pkg.go.dev/github.com/danielgtaylor/huma/v2/adapters/humago)
