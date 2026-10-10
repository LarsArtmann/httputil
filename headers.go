package httputil

// Shared canonical HTTP header-name constants. Defined once here (rather
// than per-file) so every middleware reads and writes the same names.
const (
	headerAcceptEncoding  = "Accept-Encoding"
	headerAcceptLanguage  = "Accept-Language"
	headerContentEncoding = "Content-Encoding"
	headerContentLanguage = "Content-Language"
	headerContentLength   = "Content-Length"
	headerContentRange    = "Content-Range"
	headerContentType     = "Content-Type"
	headerRange           = "Range"
	headerVary            = "Vary"
)
