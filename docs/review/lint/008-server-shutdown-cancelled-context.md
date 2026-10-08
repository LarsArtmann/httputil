# LINT-008: Server shutdown paths that cancel themselves or never drain

- **Date:** 2026-10-08
- **Severity:** Medium
- **Status:** Open
- **Consumers:** testing, reports, AI-Speed-Test, overview, github-local-sync (minor)
- **Ground truth:** httputil `Server.Shutdown` (server.go:298) applies the configured `ShutdownTimeout` only when the caller's context has no deadline; `Server.Start` is single-shot and forwards real errors on the returned channel

## What the consumers do

### testing and reports — shutdown with an already-cancelled context

`testing/internal/server/server.go:52` and the copy-paste twin
`reports/app/internal/server/server.go:88`:

```go
errChan := s.httpServer.Start()
select {
case <-ctx.Done():
	return s.Shutdown(ctx) // ctx is already Done here
...
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	...
	err := s.httpServer.Shutdown(shutdownCtx)
```

`context.WithTimeout` over a cancelled parent returns an
already-cancelled child. `http.Server.Shutdown` immediately reports
`context canceled` without waiting for in-flight requests: listeners
close (good) but every mid-flight request is killed the moment the
process proceeds to exit. The 30-second "graceful" budget is dead code
in the one path that needs it. The httputil in-fleet idiom for this is
`context.WithoutCancel(ctx)` before applying the timeout — webphone
(cmd/webphone/main.go:171) and cqrs-htmx `setup/run.go:111` both do
exactly that, with comments explaining the order of operations.

### AI-Speed-Test — no shutdown handling at all

`cmd/dashboard/main.go:1022`: `return <-server.Start()` blocks on the
error channel forever; SIGTERM kills the process with in-flight
requests outstanding. Low impact for a local dashboard, but it is
copied as an example of httputil server usage.

### overview — `ShutdownTimeout: 0`

`internal/server/server.go:157` sets `ShutdownTimeout: 0` explicitly.
httputil only applies its default (30s) when the caller passes a
deadline-less context; with zero and a deadline-less caller context,
`Shutdown` waits indefinitely. Today it works only because callers
happen to pass their own timeout; the explicit zero is a latent hang.

### github-local-sync — swallowed shutdown error

`cmd/gh-sync/main.go:441`: `_ = httpServer.Shutdown(shutdownCtx)` — the
drain result is discarded, so a timeout that killed in-flight sync
requests is invisible in logs.

## Fix

- testing/reports: `shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)`.
- AI-Speed-Test: mirror the webphone/cqrs-htmx run loop (ctx + errCh
  select + WithoutCancel shutdown).
- overview: drop the explicit zero and let `DefaultServerConfig`'s 30s
  apply, or document why infinite drain is intended.
- github-local-sync: log the shutdown error.
