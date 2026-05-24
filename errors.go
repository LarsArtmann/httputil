package httputil

import (
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Error codes for classified errors returned by ResponseRecorder operations.
// All codes use the http. namespace and are compatible with go-error-family
// for behavioral classification (Transient vs Infrastructure) and retry decisions.
const (
	// ErrCodeWriteFailed is returned when the underlying ResponseWriter.Write fails.
	// Classified as Transient (retryable).
	ErrCodeWriteFailed = "http.write_failed"

	// ErrCodeHijackUnsupported is returned when the underlying ResponseWriter
	// does not implement http.Hijacker. Classified as Infrastructure (not retryable).
	ErrCodeHijackUnsupported = "http.hijack_unsupported"

	// ErrCodeHijackFailed is returned when the underlying Hijack call fails.
	// Classified as Transient (retryable).
	ErrCodeHijackFailed = "http.hijack_failed"

	// ErrCodePushUnsupported is returned when the underlying ResponseWriter
	// does not implement http.Pusher. Classified as Infrastructure (not retryable).
	ErrCodePushUnsupported = "http.push_unsupported"

	// ErrCodePushFailed is returned when the underlying Push call fails.
	// Classified as Transient (retryable).
	ErrCodePushFailed = "http.push_failed"
)

const (
	msgRetryMaySucceed           = "This is a Transient error — retrying may succeed."
	msgInfrastructureUnsupported = "This is an Infrastructure error — the runtime environment does not support this operation."
)

// RegisterErrorClassifications maps stdlib HTTP sentinel errors to their
// behavioral families and registers error message templates for all httputil
// error codes. Call once during program startup to enable classification
// of third-party HTTP errors via errorfamily.Classify.
func RegisterErrorClassifications() {
	errorfamily.RegisterClassifications(map[error]errorfamily.Family{
		http.ErrNotSupported:    errorfamily.Infrastructure,
		http.ErrAbortHandler:    errorfamily.Transient,
		http.ErrNoCookie:        errorfamily.Transient,
		http.ErrNoLocation:      errorfamily.Transient,
		http.ErrSkipAltProtocol: errorfamily.Infrastructure,
	})

	errorfamily.RegisterTemplate(ErrCodeWriteFailed, errorfamily.MessageTemplate{
		What:   "Failed to write HTTP response body",
		Why:    "The underlying ResponseWriter.Write call returned an error (status: {{.status}}).",
		Fix:    "Check if the client disconnected or if the response buffer is full.",
		WayOut: msgRetryMaySucceed,
	})

	errorfamily.RegisterTemplate(ErrCodeHijackUnsupported, errorfamily.MessageTemplate{
		What:   "HTTP connection hijacking is not supported",
		Why:    "The underlying ResponseWriter does not implement the http.Hijacker interface.",
		Fix:    "Use a ResponseWriter that supports hijacking (e.g., net/http default writer).",
		WayOut: msgInfrastructureUnsupported,
	})

	errorfamily.RegisterTemplate(ErrCodeHijackFailed, errorfamily.MessageTemplate{
		What:   "Failed to hijack HTTP connection",
		Why:    "The underlying Hijack() call returned an error.",
		Fix:    "Check if the connection is still active and not already hijacked.",
		WayOut: msgRetryMaySucceed,
	})

	errorfamily.RegisterTemplate(ErrCodePushUnsupported, errorfamily.MessageTemplate{
		What:   "HTTP/2 server push is not supported",
		Why:    "The underlying ResponseWriter does not implement the http.Pusher interface.",
		Fix:    "Use a ResponseWriter that supports HTTP/2 push (e.g., net/http HTTP/2 writer).",
		WayOut: msgInfrastructureUnsupported,
	})

	errorfamily.RegisterTemplate(ErrCodePushFailed, errorfamily.MessageTemplate{
		What:   "Failed to push HTTP/2 resource",
		Why:    "The Push() call for target {{.target}} returned an error.",
		Fix:    "Check if the target path is valid and the connection supports HTTP/2 push.",
		WayOut: msgRetryMaySucceed,
	})
}
