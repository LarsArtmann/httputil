package httputil

import (
	"bufio"
	"hash/fnv"
	"net"
	"net/http"
	"strconv"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	headerETag        = "ETag"
	headerIfNoneMatch = "If-None-Match"
)

const (
	defaultETagMaxBufferSize = 1024 * 1024 // 1 MB.
)

// ETagConfig holds configuration for ETag generation and conditional request
// handling.
type ETagConfig struct {
	// Strength controls whether generated entity-tags are strong or weak
	// validators per RFC 7232 §2.1. The default ([EntityTagStrong]) is
	// appropriate for the built-in FNV-64a hash. Use [EntityTagWeak] if your
	// hash function may produce the same value for semantically equivalent but
	// byte-different representations.
	Strength EntityTagStrength

	// MaxBufferSize is the maximum bytes buffered for ETag computation before
	// abandoning ETag generation and streaming the response without an ETag.
	// Defaults to 1 MB. A non-positive value is clamped to the default.
	MaxBufferSize int

	// HashFunc computes the opaque-tag value from the response body. If nil,
	// FNV-64a is used (fast, 64-bit, collision-resistant for practical body
	// counts). The returned string is the unquoted opaque content; the
	// middleware wraps it with quotes and the optional weakness indicator.
	HashFunc func([]byte) string

	// SkipIfPresent controls whether the middleware respects an ETag header
	// already set by the handler. When true, the handler's ETag is used for
	// If-None-Match comparison and is not overwritten. This is recommended
	// when handlers provide their own strong validators (e.g. database
	// revision numbers). When false (default), the middleware always
	// overwrites the ETag with a body-content hash.
	SkipIfPresent bool

	// Skip, if non-nil, is called before processing each request. If it
	// returns true, the request is passed through without buffering or ETag
	// generation. Use this to exclude endpoints that are unsuitable for ETag
	// buffering (large file downloads, streaming responses, SSE, etc.).
	Skip func(*http.Request) bool

	// OnError is called when a write to the underlying ResponseWriter fails
	// after the response header has been committed and the error cannot be
	// surfaced to the client or returned from Write. If nil, such errors are
	// silently dropped (matching net/http default behavior).
	OnError func(*errorfamily.Error)
}

// DefaultETagConfig returns an ETagConfig with sensible defaults: strong
// entity-tags via FNV-64a, 1 MB buffer, no skip predicate.
func DefaultETagConfig() ETagConfig {
	return ETagConfig{
		Strength:      EntityTagStrong,
		MaxBufferSize: defaultETagMaxBufferSize,
		HashFunc:      defaultETagHashFunc,
		SkipIfPresent: false,
		Skip:          nil,
		OnError:       nil,
	}
}

// defaultETagHashFunc computes FNV-64a of data and returns the lowercase hex
// encoding. FNV-64a is a non-cryptographic hash with a 64-bit output, making
// accidental collisions astronomically unlikely for practical response-body
// counts (birthday bound: ~4 billion).
//
// The hash.Hash contract guarantees Write never returns an error; if it does,
// that signals a broken implementation and we panic with a classified
// Orchestration error.
func defaultETagHashFunc(data []byte) string {
	h := fnv.New64a()

	_, err := h.Write(data)
	if err != nil {
		panic(errorfamily.NewOrchestration(
			ErrCodeETagHashWriteFailed,
			"fnv hash.Write returned an error, violating the hash.Hash contract that Write never fails",
		))
	}

	return hexEncodeUint64(h.Sum64())
}

// Validate checks the ETagConfig for invalid values.
// Returns a *errorfamily.Error classified as Rejection on failure.
func (c ETagConfig) Validate() error {
	if c.MaxBufferSize <= 0 {
		return ErrETagConfig.WithContextf("max_buffer_size", "%d", c.MaxBufferSize)
	}

	if !c.Strength.valid() {
		return ErrETagConfig.WithContextf("strength", "%d", c.Strength)
	}

	return nil
}

// ETag returns middleware that generates ETag headers based on response body
// content and handles If-None-Match conditional requests with 304 Not
// Modified.
//
// For GET and HEAD requests, the middleware buffers the response body,
// computes an entity-tag, and compares it against the If-None-Match header
// using RFC 7232 §2.3.2 weak comparison. On match, it returns 304 Not
// Modified with no body. On mismatch, it writes the buffered body with the
// ETag header.
//
// Responses exceeding MaxBufferSize are streamed without an ETag. Hijack and
// Flush calls switch to streaming mode immediately.
func ETag(cfg ETagConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				next.ServeHTTP(resp, req)

				return
			}

			if cfg.Skip != nil && cfg.Skip(req) {
				next.ServeHTTP(resp, req)

				return
			}

			writer := newETagWriter(resp, cfg)
			next.ServeHTTP(writer, req)

			writer.flush(req)
		})
	}
}

type etagWriter struct {
	responseWrapper

	body          []byte
	strength      EntityTagStrength
	flushed       bool
	maxBufferSize int
	hashFunc      func([]byte) string
	skipIfPresent bool
	onError       func(*errorfamily.Error)
}

func newETagWriter(resp http.ResponseWriter, cfg ETagConfig) *etagWriter {
	hashFunc := cfg.HashFunc
	if hashFunc == nil {
		hashFunc = defaultETagHashFunc
	}

	// A non-positive MaxBufferSize would disable the overflow guard in Write,
	// causing unbounded buffering. Clamp to the default so a zero-value
	// ETagConfig is safe even when the caller skips Validate.
	maxBufferSize := cfg.MaxBufferSize
	if maxBufferSize <= 0 {
		maxBufferSize = defaultETagMaxBufferSize
	}

	return &etagWriter{
		responseWrapper: newResponseWrapper(resp),
		body:            nil,
		strength:        cfg.Strength,
		flushed:         false,
		maxBufferSize:   maxBufferSize,
		hashFunc:        hashFunc,
		skipIfPresent:   cfg.SkipIfPresent,
		onError:         cfg.OnError,
	}
}

// reportWriteErr wraps a post-commit write failure as a classified
// *errorfamily.Error and forwards it to the OnError callback if configured.
// The HTTP response is already in-flight, so the error cannot be surfaced
// to the client or returned from Write; this hook exists for observability
// (logging, metrics, tracing).
func (w *etagWriter) reportWriteErr(err error, message string) {
	classified := errorfamily.WrapTransient(err, ErrCodeETagWriteFailed, message)

	if w.onError != nil {
		w.onError(classified)
	}
}

func (w *etagWriter) Write(b []byte) (int, error) {
	w.writeDefaultOK()

	if w.flushed {
		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, errorfamily.WrapTransient(
				err,
				ErrCodeETagWriteFailed,
				"etag writer streaming write failed",
			)
		}

		return n, nil
	}

	if w.maxBufferSize > 0 && len(w.body)+len(b) > w.maxBufferSize {
		w.Flush()

		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, errorfamily.WrapTransient(
				err,
				ErrCodeETagWriteFailed,
				"etag writer overflow write failed",
			)
		}

		return n, nil
	}

	w.body = append(w.body, b...)

	return len(b), nil
}

func (w *etagWriter) flush(req *http.Request) {
	if w.flushed {
		return
	}

	tag := w.resolveETag()

	if tag.IsValid() {
		if w.shouldReturnNotModified(req, tag) {
			// RFC 7232 §4.1: a 304 response carries no message body, so drop
			// any body-length metadata the handler may have attached.
			w.Header().Del(headerContentLength)
			w.ResponseWriter.WriteHeader(http.StatusNotModified)
			w.headerWritten = true

			return
		}
	}

	// RFC 7230 §3.3: HEAD responses MUST NOT include a message body. The
	// buffered bytes still let us advertise an accurate Content-Length so a
	// client can size the representation without transferring it.
	if req.Method == http.MethodHead {
		w.Header().Set(headerContentLength, strconv.Itoa(len(w.body)))
	}

	w.markFlushed()

	w.writeHeaderToUnderlying()

	if req.Method == http.MethodHead {
		return
	}

	// Post-header-commit body writes are fundamentally unreportable: the
	// handler has returned and the HTTP response is already in-flight.
	// The error is classified and forwarded to OnError for observability.
	if len(w.body) > 0 {
		_, err := w.ResponseWriter.Write(w.body)
		if err != nil {
			w.reportWriteErr(err, "etag flush body write failed")
		}
	}
}

// resolveETag determines which entity-tag to use for this response. If
// SkipIfPresent is configured and the handler already set a valid ETag
// header, that value is respected. Otherwise, a new entity-tag is computed
// from the buffered body and set on the response header.
func (w *etagWriter) resolveETag() EntityTag {
	if w.skipIfPresent {
		existing := w.Header().Get(headerETag)
		if existing != "" {
			if tag, ok := ParseEntityTag(existing); ok {
				return tag
			}
		}
	}

	tag := w.computeETag()

	if tag.IsValid() {
		w.Header().Set(headerETag, tag.String())
	}

	return tag
}

func (w *etagWriter) shouldReturnNotModified(req *http.Request, tag EntityTag) bool {
	if !w.isCacheableStatus() {
		return false
	}

	inm := strings.Join(req.Header.Values(headerIfNoneMatch), ", ")
	if inm == "" {
		return false
	}

	return MatchesIfNoneMatch(tag, inm)
}

func (w *etagWriter) computeETag() EntityTag {
	if len(w.body) == 0 && !w.wroteHeader {
		return EntityTag{} //nolint:exhaustruct // zero value signals no ETag
	}

	opaque := w.hashFunc(w.body)

	return NewEntityTag(opaque, w.strength)
}

func (w *etagWriter) isCacheableStatus() bool {
	return w.status == 0 || (w.status >= http.StatusOK && w.status < http.StatusMultipleChoices)
}

func (w *etagWriter) Flush() {
	if w.flushed {
		w.responseWrapper.Flush()

		return
	}

	w.markFlushed()

	w.writeHeaderToUnderlying()

	// Post-header-commit body writes are fundamentally unreportable: the
	// handler has returned and the HTTP response is already in-flight.
	// The error is classified and forwarded to OnError for observability.
	if len(w.body) > 0 {
		_, err := w.ResponseWriter.Write(w.body)
		if err != nil {
			w.reportWriteErr(err, "etag flush body write failed")
		}

		w.body = w.body[:0]
	}

	w.responseWrapper.Flush()
}

func (w *etagWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.markFlushed()

	w.writeHeaderToUnderlying()

	return w.responseWrapper.Hijack()
}

// markFlushed transitions the writer into streaming mode by setting the
// flushed flag. The buffered status header is committed separately via
// writeHeaderToUnderlying.
func (w *etagWriter) markFlushed() {
	w.flushed = true
}
