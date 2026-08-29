package httputil

// Shared canonical HTTP header-name constants. Defined once here (rather
// than per-file) so every middleware reads and writes the same names.
const (
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentLength   = "Content-Length"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"
)
