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

	// ErrCodeCompressWriteFailed is returned when gzip write during
	// compression fails. Classified as Transient (retryable).
	ErrCodeCompressWriteFailed = "http.compress_write_failed"

	// ErrCodeETagWriteFailed is returned when the ETag writer fails to
	// write buffered or streamed data. Classified as Transient (retryable).
	ErrCodeETagWriteFailed = "http.etag_write_failed"
)

const (
	msgRetryMaySucceed           = "This is a Transient error — retrying may succeed."
	msgInfrastructureUnsupported = "This is an Infrastructure error — the runtime environment does not support this operation."
)

func registerErrorTemplate(code, what, why, fix, wayOut string) {
	errorfamily.RegisterTemplate(code, errorfamily.MessageTemplate{
		What:   what,
		Why:    why,
		Fix:    fix,
		WayOut: wayOut,
	})
}

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

	registerAllErrorTemplates()
}

func registerAllErrorTemplates() {
	registerErrorTemplate(
		ErrCodeWriteFailed,
		"Failed to write HTTP response body",
		"The underlying ResponseWriter.Write call returned an error (status: {{.status}}).",
		"Check if the client disconnected or if the response buffer is full.",
		msgRetryMaySucceed,
	)

	registerErrorTemplate(
		ErrCodeHijackUnsupported,
		"HTTP connection hijacking is not supported",
		"The underlying ResponseWriter does not implement the http.Hijacker interface.",
		"Use a ResponseWriter that supports hijacking (e.g., net/http default writer).",
		msgInfrastructureUnsupported,
	)

	registerErrorTemplate(
		ErrCodeHijackFailed,
		"Failed to hijack HTTP connection",
		"The underlying Hijack() call returned an error.",
		"Check if the connection is still active and not already hijacked.",
		msgRetryMaySucceed,
	)

	registerErrorTemplate(
		ErrCodePushUnsupported,
		"HTTP/2 server push is not supported",
		"The underlying ResponseWriter does not implement the http.Pusher interface.",
		"Use a ResponseWriter that supports HTTP/2 push (e.g., net/http HTTP/2 writer).",
		msgInfrastructureUnsupported,
	)

	registerErrorTemplate(
		ErrCodePushFailed,
		"Failed to push HTTP/2 resource",
		"The Push() call for target {{.target}} returned an error.",
		"Check if the target path is valid and the connection supports HTTP/2 push.",
		msgRetryMaySucceed,
	)

	registerErrorTemplate(
		ErrCodeCompressWriteFailed,
		"Failed to write compressed HTTP response",
		"The gzip writer or underlying ResponseWriter returned an error during compression.",
		"Check if the client disconnected or if the response buffer is full.",
		msgRetryMaySucceed,
	)

	registerErrorTemplate(
		ErrCodeETagWriteFailed,
		"Failed to write ETag-buffered HTTP response",
		"The underlying ResponseWriter.Write call returned an error while streaming ETag data.",
		"Check if the client disconnected or if the response buffer is full.",
		msgRetryMaySucceed,
	)
}
