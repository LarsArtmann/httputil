package httputil

import "net/http"

// Capabilities describes which optional http.ResponseWriter interfaces
// the underlying writer supports.
type Capabilities struct {
	// Hijacker is true when the writer implements http.Hijacker, allowing
	// low-level connection takeover for WebSocket and similar upgrades.
	Hijacker bool

	// Flusher is true when the writer implements http.Flusher, allowing
	// immediate flushing of buffered response data to the client.
	Flusher bool
}

// DetectCapabilities inspects w and reports which optional ResponseWriter
// interfaces it supports. Useful when composing middleware that needs to know
// whether operations like Hijack or Flush are available before attempting them.
//
// Audit decision (2026-08-29): kept. It is not used by the middleware
// internals (each wrapper probes interfaces directly where needed), but it is
// exported public API for consumers composing their own ResponseWriter
// wrappers, classified Frozen in docs/v1-stability.md, and removing it would
// be a breaking change with no upside.
func DetectCapabilities(w http.ResponseWriter) Capabilities {
	_, hijacker := w.(http.Hijacker)
	_, flusher := w.(http.Flusher)

	return Capabilities{
		Hijacker: hijacker,
		Flusher:  flusher,
	}
}
