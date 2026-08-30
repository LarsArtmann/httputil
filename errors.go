package httputil

import (
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
	etag "github.com/larsartmann/go-etag/server"
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

	// ErrCodeCompressWriteFailed is returned when gzip write during
	// compression fails. Classified as Transient (retryable).
	ErrCodeCompressWriteFailed = "http.compress_write_failed"
)

// Typed mirrors of the exported untyped string codes above, used for
// internal construction via the Code constructor and Wrap methods. The
// exported constants remain untyped strings for backward compatibility;
// these typed constants keep internal call sites compile-time grouped.
const (
	codeWriteFailed         = Code(ErrCodeWriteFailed)
	codeHijackUnsupported   = Code(ErrCodeHijackUnsupported)
	codeHijackFailed        = Code(ErrCodeHijackFailed)
	codeCompressWriteFailed = Code(ErrCodeCompressWriteFailed)
)

const (
	msgRetryMaySucceed           = "This is a Transient error — retrying may succeed."
	msgInfrastructureUnsupported = "This is an Infrastructure error — the runtime environment does not support this operation."
	msgCheckDisconnected         = "Check if the client disconnected or if the response buffer is full."
	msgServerTimeoutDefaults     = "DefaultServerConfig() ships valid timeouts."
	msgQValueDiagnosticFix       = "No action needed: malformed q-values fall back to the default priority during negotiation."
	msgQValueDiagnosticWayOut    = "This is diagnostic only; the request is still served."
)

// errorTemplates maps every error code this package can produce (plus the
// go-etag codes it registers on behalf of) to its user-facing message
// template. Templates use {key} placeholders filled from the error's context.
// The completeness test in errors_templates_test.go asserts every code in
// allHTTputilErrorCodes has an entry here.
//
//nolint:gochecknoglobals // immutable data table: populated once, never mutated
var errorTemplates = map[string]errorfamily.MessageTemplate{
	ErrCodeWriteFailed: {
		What:   "Failed to write HTTP response body",
		Why:    "The underlying ResponseWriter.Write call returned an error (status: {{.status}}).",
		Fix:    msgCheckDisconnected,
		WayOut: msgRetryMaySucceed,
	},
	ErrCodeHijackUnsupported: {
		What:   "HTTP connection hijacking is not supported",
		Why:    "The underlying ResponseWriter does not implement the http.Hijacker interface.",
		Fix:    "Use a ResponseWriter that supports hijacking (e.g., net/http default writer).",
		WayOut: msgInfrastructureUnsupported,
	},
	ErrCodeHijackFailed: {
		What:   "Failed to hijack HTTP connection",
		Why:    "The underlying Hijack() call returned an error.",
		Fix:    "Check if the connection is still active and not already hijacked.",
		WayOut: msgRetryMaySucceed,
	},
	ErrCodeCompressWriteFailed: {
		What:   "Failed to write compressed HTTP response",
		Why:    "The gzip writer or underlying ResponseWriter returned an error during compression.",
		Fix:    msgCheckDisconnected,
		WayOut: msgRetryMaySucceed,
	},
	etag.ErrCodeETagWriteFailed: {
		What:   "Failed to write ETag-buffered HTTP response",
		Why:    "The underlying ResponseWriter.Write call returned an error while streaming ETag data.",
		Fix:    msgCheckDisconnected,
		WayOut: msgRetryMaySucceed,
	},
	etag.ErrCodeInvalidConfig: {
		What:   "ETag configuration is invalid",
		Why:    "One or more fields of ETagConfig have invalid values.",
		Fix:    "Review the ETagConfig field values and ensure MaxBufferSize is positive.",
		WayOut: "Check your ETagConfig values and try again.",
	},
	etag.ErrCodeHashWriteFailed: {
		What:   "Hash function failed to accept data",
		Why:    "The hash.Write call returned an error, which violates the hash.Hash contract that Write never fails.",
		Fix:    "This indicates a bug in the hash implementation. Report it to the library author.",
		WayOut: "This is likely a bug. Please report it if the problem persists.",
	},

	// CSRF. The legacy exported codes use the historical underscore spelling.
	string(codeCSRFInvalid): {
		What:   "CSRF token is missing or invalid",
		Why:    "The request failed double-submit cookie validation: the token was absent, malformed, or did not match the csrf cookie.",
		Fix:    "Ensure forms include the CSRF field and HTMX requests send the CSRF header with the current token.",
		WayOut: "Fetch a fresh page to obtain a new token and cookie.",
	},
	string(codeCSRFConfig): {
		What:   "CSRF configuration is invalid",
		Why:    "A CSRFConfig validation error occurred; the specific field is in the cause chain.",
		Fix:    "Fix the configuration issue named in the cause error.",
		WayOut: "Start from CSRFConfig{} which uses secure defaults.",
	},
	string(codeCSRFSameSiteInsecure): {
		What:   "CSRF cookie would be sent insecurely",
		Why:    "CSRFConfig has SameSite=None without Secure=true; browsers reject SameSite=None cookies on non-HTTPS connections.",
		Fix:    "Set Secure=true (recommended) or use SameSite=Lax/Strict.",
		WayOut: "SameSite=Lax keeps most single-sign-in flows working.",
	},
	string(codeCSRFUnsafeOrigin): {
		What:   "CSRF trusted origin list contains an unsafe entry",
		Why:    "CSRFConfig.TrustedOrigins contains {origin}; wildcard or empty entries would trust every origin and defeat CSRF protection.",
		Fix:    "List specific origins such as https://app.example.com.",
		WayOut: "Remove the entry entirely to trust no additional origins.",
	},
	string(codeCSRFUnsafeProxy): {
		What:   "CSRF trusted proxy list contains an empty entry",
		Why:    "CSRFConfig.TrustedProxies has an empty string, which would match no proxy and signals a config mistake.",
		Fix:    "Remove the empty entry or set TrustedProxies to nil to trust no proxies.",
		WayOut: "DefaultCSRFConfig trusts only the direct connection.",
	},
	string(codeCSRFInvalidCIDR): {
		What:   "CSRF trusted proxy entry is not a valid CIDR",
		Why:    "CSRFConfig.TrustedProxies contains {proxy}, which failed CIDR parsing: {parse_error}.",
		Fix:    "Use CIDR notation such as 10.0.0.0/8, or a bare IP such as 10.0.0.1.",
		WayOut: "Fix the entry; the middleware refuses to start with an unparseable proxy list.",
	},

	// CORS config (Rejection family: fix the configuration, never retry).
	string(codeCorsCredentialsWithAllOrigins): {
		What:   "CORS configuration combines credentials with a wildcard origin",
		Why:    "CORSConfig has AllowCredentials=true and AllowAllOrigins=true, which the CORS spec forbids: browsers reject Access-Control-Allow-Origin: * when credentials are enabled.",
		Fix:    "Set AllowAllOrigins=false and list specific origins in AllowedOrigins, or disable AllowCredentials.",
		WayOut: "Start from DefaultCORSConfig() and change one field at a time.",
	},
	string(codeCorsMaxAgeNegative): {
		What:   "CORS preflight cache duration is negative",
		Why:    "CORSConfig.MaxAge is {max_age}; preflight cache durations cannot be negative.",
		Fix:    "Set MaxAge to zero (no caching) or a positive number of seconds.",
		WayOut: "DefaultCORSConfig() ships a valid MaxAge.",
	},

	// Server config.
	string(codeServerAddrEmpty): {
		What:   "Server listen address is empty",
		Why:    "ServerConfig.Addr is empty, so the server would bind to a random port, which is almost never intended.",
		Fix:    "Set Addr to an explicit address such as \":8080\" or \":http\".",
		WayOut: "DefaultServerConfig() provides \":8080\".",
	},
	string(codeServerReadTimeoutNegative): {
		What:   "Server read timeout is negative",
		Why:    "ServerConfig.ReadTimeout is {read_timeout}; timeouts cannot be negative.",
		Fix:    "Set ReadTimeout to zero (no timeout) or a positive duration.",
		WayOut: msgServerTimeoutDefaults,
	},
	string(codeServerReadHeaderTimeoutNegative): {
		What:   "Server read-header timeout is negative",
		Why:    "ServerConfig.ReadHeaderTimeout is {read_header_timeout}; timeouts cannot be negative.",
		Fix:    "Set ReadHeaderTimeout to zero (no timeout) or a positive duration.",
		WayOut: msgServerTimeoutDefaults,
	},
	string(codeServerWriteTimeoutNegative): {
		What:   "Server write timeout is negative",
		Why:    "ServerConfig.WriteTimeout is {write_timeout}; timeouts cannot be negative.",
		Fix:    "Set WriteTimeout to zero (no timeout) or a positive duration.",
		WayOut: msgServerTimeoutDefaults,
	},
	string(codeServerIdleTimeoutNegative): {
		What:   "Server idle timeout is negative",
		Why:    "ServerConfig.IdleTimeout is {idle_timeout}; timeouts cannot be negative.",
		Fix:    "Set IdleTimeout to zero (no timeout) or a positive duration.",
		WayOut: msgServerTimeoutDefaults,
	},
	string(codeServerShutdownTimeoutNegative): {
		What:   "Server shutdown timeout is negative",
		Why:    "ServerConfig.ShutdownTimeout is {shutdown_timeout}; timeouts cannot be negative.",
		Fix:    "Set ShutdownTimeout to zero (no timeout) or a positive duration.",
		WayOut: msgServerTimeoutDefaults,
	},
	string(codeServerTimeoutOrdering): {
		What:   "Server header-read timeout exceeds the full read timeout",
		Why:    "ServerConfig.ReadHeaderTimeout ({read_header_timeout}) is greater than ReadTimeout ({read_timeout}); RFC 7230 section 6 requires the header timeout to fit within the full read timeout.",
		Fix:    "Set ReadHeaderTimeout to a value <= ReadTimeout, or increase ReadTimeout.",
		WayOut: "DefaultServerConfig() uses 5s header / 10s full read.",
	},
	string(codeServerTLSMinVersionInsecure): {
		What:   "TLS minimum version is below TLS 1.2",
		Why:    "ServerConfig.TLSConfig.MinVersion is {min_version}; RFC 8996 deprecates TLS 1.0 and 1.1.",
		Fix:    "Set MinVersion to tls.VersionTLS12 or higher, or leave it zero (Go defaults to TLS 1.2).",
		WayOut: "Test with a modern client; old clients will fail the handshake by design.",
	},

	// Compression config and Accept-Encoding q-value parsing.
	string(codeCompressionLevelInvalid): {
		What:   "Compression level is out of range",
		Why:    "CompressionConfig.Level is {level}; valid levels are gzip.HuffmanOnly through gzip.BestCompression, plus gzip.DefaultCompression.",
		Fix:    "Set Level to gzip.DefaultCompression or a value in the valid range.",
		WayOut: "DefaultCompressionConfig() ships gzip.DefaultCompression.",
	},
	string(codeCompressionMinSizeNeg): {
		What:   "Compression minimum size is negative",
		Why:    "CompressionConfig.MinSize is {min_size}; a negative threshold is meaningless.",
		Fix:    "Set MinSize to zero (compress everything) or a positive byte count.",
		WayOut: "DefaultCompressionConfig() ships a sensible threshold.",
	},
	string(codeCompressionNoFactory): {
		What:   "Compression has no writer factories",
		Why:    "CompressionConfig.WriterFactories is empty, so no encoding could ever be produced.",
		Fix:    "Set WriterFactories via DefaultWriterFactories(), or leave it empty when constructing through Compression() so defaults are filled in.",
		WayOut: "DefaultCompressionConfig() ships gzip and deflate factories.",
	},
	string(codeCompressionQValueEmpty): {
		What:   "Accept-Encoding q-value is empty",
		Why:    "A q-value parameter in the Accept-Encoding header is empty where a number was expected.",
		Fix:    msgQValueDiagnosticFix,
		WayOut: msgQValueDiagnosticWayOut,
	},
	string(codeCompressionQValueInvalid): {
		What:   "Accept-Encoding q-value is malformed",
		Why:    "The q-value {input} does not start with 0 or 1 as RFC 7231 requires.",
		Fix:    msgQValueDiagnosticFix,
		WayOut: msgQValueDiagnosticWayOut,
	},
	string(codeCompressionQValueTrail): {
		What:   "Accept-Encoding q-value has trailing characters",
		Why:    "The q-value {input} has characters after the third decimal digit.",
		Fix:    msgQValueDiagnosticFix,
		WayOut: msgQValueDiagnosticWayOut,
	},
	string(codeCompressionQValueTooBig): {
		What:   "Accept-Encoding q-value exceeds 1.0",
		Why:    "The q-value {input} is greater than the RFC 7231 maximum of 1.0.",
		Fix:    msgQValueDiagnosticFix,
		WayOut: msgQValueDiagnosticWayOut,
	},

	// Rate limiting config (keyed and deprecated).
	string(codeRatelimitKeyedLimitZero): {
		What:   "Rate limit is not positive",
		Why:    "KeyedRateLimiterConfig.Limit is zero; a limit of zero requests would reject everything.",
		Fix:    "Set Limit to a positive number of requests per Window.",
		WayOut: "DefaultKeyedRateLimiterConfig() ships valid values.",
	},
	string(codeRatelimitKeyedWindowZero): {
		What:   "Rate limit window is not positive",
		Why:    "KeyedRateLimiterConfig.Window is {window}; the refill window must be a positive duration.",
		Fix:    "Set Window to a positive duration such as time.Minute.",
		WayOut: "DefaultKeyedRateLimiterConfig() ships valid values.",
	},
	string(codeRatelimitKeyedTTLNegative): {
		What:   "Rate limit eviction TTL is negative",
		Why:    "KeyedRateLimiterConfig.TTL is {ttl}; zero disables eviction, negative is invalid.",
		Fix:    "Set TTL to zero (keep idle keys forever) or a positive duration.",
		WayOut: "Leave TTL at zero for small, bounded key populations.",
	},
	string(codeRatelimitNilLimiter): {
		What:   "Rate limiter is missing",
		Why:    "RateLimitConfig.Limiter is nil, so no rate decision could be made.",
		Fix:    "Set Limiter to a RateLimiter such as NewTokenBucketLimiter, or migrate to KeyedRateLimiterMiddleware.",
		WayOut: "KeyedRateLimiterMiddleware supersedes this deprecated API.",
	},
	string(codeRatelimitInvalidRate): {
		What:   "Token bucket rate is not positive",
		Why:    "The rate must be a positive number of tokens per second.",
		Fix:    "Pass a positive rate to NewTokenBucketLimiter.",
		WayOut: "Consider KeyedRateLimiterMiddleware instead of the deprecated API.",
	},
	string(codeRatelimitInvalidBurst): {
		What:   "Token bucket burst is not positive",
		Why:    "The burst must be a positive number of tokens.",
		Fix:    "Pass a positive burst to NewTokenBucketLimiter.",
		WayOut: "Consider KeyedRateLimiterMiddleware instead of the deprecated API.",
	},
	string(codeRatelimitInvalidStatus): {
		What:   "Rate limit denial status is not a valid HTTP status code",
		Why:    "RateLimitConfig.Status is {status}; status codes must be in the 100-599 range or zero for the default.",
		Fix:    "Set Status to zero (429 Too Many Requests) or a valid status code.",
		WayOut: "The default denial status is 429.",
	},

	// Per-middleware config.
	string(codeMaxBodySizeNegative): {
		What:   "Request body size limit is negative",
		Why:    "MaxBodySizeConfig.MaxBytes is {max_bytes}; a negative limit is always a bug.",
		Fix:    "Set MaxBytes to a positive byte count; zero imposes a zero-byte limit that rejects any non-empty body.",
		WayOut: "DefaultMaxBodySizeConfig() ships a 1 MiB limit.",
	},
	string(codeRequestIDNilGenerateID): {
		What:   "Request ID generator is missing",
		Why:    "RequestIDConfig.GenerateID is nil, so no request ID could be produced.",
		Fix:    "Set GenerateID to a function such as the default generateTimeOrderedID, or start from DefaultRequestIDConfig().",
		WayOut: "DefaultRequestIDConfig() wires a time-ordered generator.",
	},
	string(codeRequestIDEmptyResponseHeader): {
		What:   "Request ID response header name is empty",
		Why:    "RequestIDConfig.ResponseHeader is empty; responses would carry no ID header.",
		Fix:    "Set ResponseHeader to a header name such as X-Request-Id.",
		WayOut: "DefaultRequestIDConfig() uses X-Request-Id.",
	},
	string(codeRequestIDEmptyIncomingHeader): {
		What:   "Request ID incoming header name is empty",
		Why:    "RequestIDConfig.IncomingHeader is empty; incoming IDs could not be propagated.",
		Fix:    "Set IncomingHeader to a header name such as X-Request-Id.",
		WayOut: "DefaultRequestIDConfig() uses X-Request-Id.",
	},
	string(codeSecurityInvalidFrameOptions): {
		What:   "Frame Options value is invalid",
		Why:    "SecurityHeadersConfig.FrameOptions is {frame_options}; valid values are DENY, SAMEORIGIN, SecurityHeaderSkip, or empty.",
		Fix:    "Set FrameOptions to one of the valid values or leave it empty to send no header.",
		WayOut: "DefaultSecurityHeadersConfig() uses SAMEORIGIN.",
	},
	string(codeMetricsNilRecorder): {
		What:   "Metrics recorder is missing",
		Why:    "MetricsConfig.Recorder is nil, so request metrics would have nowhere to go.",
		Fix:    "Set Recorder to an implementation of MetricsRecorder that is safe for concurrent use.",
		WayOut: "DefaultMetricsConfig() documents the expected shape.",
	},
	string(codeNonceTooSmall): {
		What:   "CSP nonce size is below the minimum",
		Why:    "NonceConfig.Size is {size} bytes; nonces shorter than 16 bytes (128 bits) are guessable and violate the CSP Level 3 recommendation.",
		Fix:    "Set Size to zero (use the 20-byte default) or at least 16.",
		WayOut: "DefaultNonceConfig() ships a secure size.",
	},
	string(codeStackDuplicateMiddleware): {
		What:   "Middleware name is already in the stack",
		Why:    "MiddlewareStack.Add was called twice with the name {name}; duplicate names would break ordering validation.",
		Fix:    "Use a unique name for each middleware entry.",
		WayOut: "Use the Middleware* name constants for well-known middleware.",
	},
	string(codeStackRecoveryNotFirst): {
		What:   "Recovery middleware is not the outermost middleware",
		Why:    "Recovery was found at stack position {position} instead of 0; it must wrap all other middleware to catch their panics.",
		Fix:    "Add Recovery with MiddlewareRecovery first, or reorder the stack so it is outermost.",
		WayOut: "MiddlewareStack.Validate() reports this before any request is served.",
	},
	string(codeDecompressionSizeNegative): {
		What:   "Decompression size limit is negative",
		Why:    "DecompressionConfig.MaxDecompressionSize is {max_decompression_size}; a negative limit is meaningless.",
		Fix:    "Set MaxDecompressionSize to zero (use the 16 MiB default) or a positive byte count.",
		WayOut: "DefaultDecompressionConfig() ships a 16 MiB bomb-protection limit.",
	},

	// Runtime errors.
	string(codeServerShutdownFailed): {
		What:   "HTTP server shutdown failed",
		Why:    "The underlying http.Server.Shutdown call failed, typically because the shutdown context expired while connections were still open.",
		Fix:    "Extend ShutdownTimeout, drain in-flight requests faster, or investigate the connection named in the cause.",
		WayOut: "The process state is still controlled; shutdown can be re-attempted.",
	},
	string(codeDecompressionSizeExceeded): {
		What:   "Decompressed request body exceeded the size limit",
		Why:    "The decompressed body grew past the configured MaxDecompressionSize; this is the decompression-bomb protection rejecting the request.",
		Fix:    "Do not retry: the same body will fail again. Raise MaxDecompressionSize only if legitimate payloads are larger.",
		WayOut: "The request was rejected before your handler ran; the connection is still usable.",
	},
	string(codeDecompressionReadFailed): {
		What:   "Decompressing the request body failed",
		Why:    "The compressed request body could not be decoded; it is corrupt or truncated.",
		Fix:    "Do not retry with the same body. The client should re-upload a valid gzip or deflate payload.",
		WayOut: "The handler never saw the body; respond with an appropriate client error.",
	},
	string(codeDecompressionCloseFailed): {
		What:   "Closing the decompressed request body failed",
		Why:    "The underlying decompressor or body reader failed to close cleanly.",
		Fix:    "The response is usually already sent; log and investigate if it recurs.",
		WayOut: "This is cleanup-only; the request itself has completed.",
	},
	string(codeCompressionPoolTypeUnexpected): {
		What:   "Compression writer pool returned an unexpected type",
		Why:    "A writer pool yielded an element of type {pool_element_type} that does not satisfy io.WriteCloser; the pool and the WriterFactory disagree on the writer type.",
		Fix:    "This is a bug in a custom WriterFactory: ensure the factory always returns an io.WriteCloser.",
		WayOut: "Use DefaultWriterFactories() or report the bug to the factory author.",
	},
}

// RegisterErrorClassifications maps stdlib HTTP sentinel errors to their
// behavioral families and registers error message templates for all httputil
// and go-etag error codes. Call once during program startup to enable
// classification of HTTP errors via errorfamily.Classify.
//
// This registers a strict superset of go-etag's error templates and stdlib
// classifications, so consumers need only call this once. Do not also call
// etag.RegisterErrorClassifications.
func RegisterErrorClassifications() {
	errorfamily.RegisterClassifications(map[error]errorfamily.Family{
		http.ErrNotSupported:    errorfamily.Infrastructure,
		http.ErrAbortHandler:    errorfamily.Transient,
		http.ErrNoCookie:        errorfamily.Transient,
		http.ErrNoLocation:      errorfamily.Transient,
		http.ErrSkipAltProtocol: errorfamily.Infrastructure,
	})

	for code, tmpl := range errorTemplates {
		errorfamily.RegisterTemplate(code, tmpl)
	}
}
