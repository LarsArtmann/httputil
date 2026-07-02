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
func DetectCapabilities(w http.ResponseWriter) Capabilities {
	_, hijacker := w.(http.Hijacker)
	_, flusher := w.(http.Flusher)

	return Capabilities{
		Hijacker: hijacker,
		Flusher:  flusher,
	}
}
