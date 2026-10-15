# Shutdown-Drain Settlement — Decision Note (WaitGroup vs documented-race)

_Created: 2026-10-15 18:40 CEST (pareto T23) · Sources: ROADMAP "Server shutdown-drain semantics" (routed from the archived `2026-09-11_09-01` g1/f1/f5), `server.go:210-328` read this session · Audience: the owner · Decision needed: ship settlement or re-document the window._

## The precise current behavior (verified in `server.go` this session)

1. `Start`/`StartTLS` spawn the Serve goroutine. When `httpServer.Serve(listener)` returns, the goroutine clears the listener and `started` (server.go:232-241).
2. `Shutdown(ctx)` waits for `httpServer.Shutdown(ctx)` — which closes the listeners first, then waits for in-flight requests (or ctx expiry) — and then `clearListener()` (server.go:312-328). It does **not** clear `started` and does **not** wait for the Serve goroutine.
3. `http.Server.Serve` returns **as soon as the listener closes**, which happens at the START of `httpServer.Shutdown` — before in-flight requests drain.

## The race, stated exactly

`Serve` returns `http.ErrServerClosed` the moment the listener closes (early in `Shutdown`), so the goroutine flips `started` to **false** while the old `http.Server` may still be finishing in-flight handlers. A `Start()` issued in that window passes the CAS and binds a new listener on the now-free port. Two `http.Server`s are then live: the new one accepting, the old one finishing its tail (or, past a timed-out `Shutdown`, an unbounded stuck handler still writing). The public godoc currently promises "retrying the same Server after … `Shutdown()` is safe" — the code delivers restart-safety-with-a-tail-overlap, slightly weaker than documented.

## Option A — WaitGroup settlement (recommended)

Serve goroutine: `defer srv.serveDone.Done()` after the cleanup lines; `Shutdown`: `srv.serveDone.Wait()` before returning. (Unexported `sync.WaitGroup` field — additive, zero value usable, no API surface change.)

- **Postcondition becomes true:** `Shutdown` returning ⟹ listener cleared, `started` false, Serve goroutine finished ⟹ `Start` is safe with zero overlap.
- **Cost:** one goroutine-scheduling hop on the Shutdown path (microseconds); the existing `ShutdownTimeout` context already bounds the drain, and the WG wait adds no new hang class (the Serve goroutine cannot block after `Serve` returns — cleanup is two atomic/mutex operations).
- **Compatibility:** none broken; tests that observe `started`/`ListenerAddr` after `Shutdown` see strictly-settled state (the existing settlement tests keep passing, now trivially).

## Option B — keep the window, document it precisely

Reword the godoc/README guarantee to "restart-safe except for a brief post-Shutdown window while the previous instance's in-flight handlers finish; handlers that outlive `ShutdownTimeout` keep serving during that window." Cost: the guarantee stays conditional and every future doc reader re-derives the hazard; the 2026-10-10 settlement work (which documented the error-path guarantee) would carry an asterisk.

## Recommendation

**A.** It converts a documented hazard into a verified invariant for ~5 lines, matches the guarantee the godoc already states, and closes the last unsettled lifecycle edge (the failure paths were settled in the 2026-10-10 batch; this is the shutdown-path twin). If the owner prefers B, the default-if-unanswered is **A in the next patch window**, because the current godoc already promises what A delivers — B would be a doc regression against the shipped promise.
