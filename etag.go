package httputil

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	headerETag        = "ETag"
	headerIfNoneMatch = "If-None-Match"
)

const (
	hashUint64Bytes          = 8
	hashUint64HexSize        = 16
	etagWeakLen              = hashUint64HexSize + 4 // W/"" + hex + "
	etagStrongLen            = hashUint64HexSize + 2 // "" + hex + "
	defaultETagMaxBufferSize = 1024 * 1024           // 1 MB
)

// ETagConfig holds configuration for ETag generation.
type ETagConfig struct {
	Weak bool
	// MaxBufferSize is the maximum bytes buffered for ETag computation
	// before abandoning ETag generation and streaming the response.
	MaxBufferSize int
	// HashFunc computes a 64-bit hash of the response body for ETag
	// generation. If nil, FNV-64a is used (fast, 64-bit, collision-resistant
	// for practical body counts). Provide a custom function for
	// application-specific hashing needs.
	HashFunc func([]byte) uint64
}

// DefaultETagConfig returns an ETagConfig with sensible defaults.
func DefaultETagConfig() ETagConfig {
	return ETagConfig{
		Weak:          false,
		MaxBufferSize: defaultETagMaxBufferSize,
		HashFunc:      defaultETagHash,
	}
}

// defaultETagHash computes FNV-64a of data. FNV-64a is a non-cryptographic
// hash with a 64-bit output, making accidental collisions astronomically
// unlikely for practical response-body counts (birthday bound: ~4 billion).
func defaultETagHash(data []byte) uint64 {
	h := fnv.New64a()

	_, _ = h.Write(data)

	return h.Sum64()
}

var errNonPositiveMaxBufferSize = errors.New("ETagConfig.MaxBufferSize must be positive")

// Validate checks the ETagConfig for invalid values.
func (c ETagConfig) Validate() error {
	if c.MaxBufferSize <= 0 {
		return fmt.Errorf("%w: got %d", errNonPositiveMaxBufferSize, c.MaxBufferSize)
	}

	return nil
}

// ETag returns middleware that generates ETag headers based on response body
// content and handles If-None-Match conditional requests with 304 Not Modified.
func ETag(cfg ETagConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				next.ServeHTTP(resp, req)

				return
			}

			ew := newETagWriter(resp, cfg)
			next.ServeHTTP(ew, req)

			ew.flush(req)
		})
	}
}

type etagWriter struct {
	responseWrapper

	body          []byte
	weak          bool
	flushed       bool
	maxBufferSize int
	hashFunc      func([]byte) uint64
}

func newETagWriter(resp http.ResponseWriter, cfg ETagConfig) *etagWriter {
	hashFunc := cfg.HashFunc
	if hashFunc == nil {
		hashFunc = defaultETagHash
	}

	return &etagWriter{
		responseWrapper: newResponseWrapper(resp),
		body:            nil,
		weak:            cfg.Weak,
		flushed:         false,
		maxBufferSize:   cfg.MaxBufferSize,
		hashFunc:        hashFunc,
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

	etag := w.computeETag()

	if etag != "" {
		w.Header().Set(headerETag, etag)

		if w.matchesIfNoneMatch(req, etag) && w.isCacheableStatus() {
			w.ResponseWriter.WriteHeader(http.StatusNotModified)
			w.headerWritten = true

			return
		}
	}

	w.writeHeaderToUnderlying()

	if len(w.body) > 0 {
		_, _ = w.ResponseWriter.Write(w.body)
	}
}

func (w *etagWriter) computeETag() string {
	if len(w.body) == 0 && !w.wroteHeader {
		return ""
	}

	hash := w.hashFunc(w.body)

	var buf [hashUint64Bytes]byte
	binary.BigEndian.PutUint64(buf[:], hash)

	if w.weak {
		var etag [etagWeakLen]byte

		etag[0] = 'W'
		etag[1] = '/'
		etag[2] = '"'

		encodeHex(etag[3:], buf[:])

		etag[etagWeakLen-1] = '"'

		return string(etag[:])
	}

	var etag [etagStrongLen]byte

	etag[0] = '"'

	encodeHex(etag[1:], buf[:])

	etag[etagStrongLen-1] = '"'

	return string(etag[:])
}

// encodeHex writes the hex encoding of src into dst. dst must have length >= 2*len(src).
func encodeHex(dst, src []byte) {
	for i, b := range src {
		dst[i*2] = hexDigitsLower[b>>4]
		dst[i*2+1] = hexDigitsLower[b&0x0f]
	}
}

func (w *etagWriter) matchesIfNoneMatch(req *http.Request, etag string) bool {
	inm := req.Header.Get(headerIfNoneMatch)
	if inm == "*" {
		return true
	}

	return etagInList(inm, etag)
}

// parseETagList splits a comma-separated list of entity-tags, respecting
// commas inside quoted opaque-tags per RFC 7232 §2.3 (etagc permits any
// VCHAR except DQUOTE, which includes comma). Backslash escapes inside
// quoted-strings are honored so that an escaped DQUOTE does not toggle
// the quote state.
func parseETagList(list string) []string {
	var tags []string

	start := 0

	inQuotes := false

	escaped := false

	for i := range list {
		if escaped {
			escaped = false

			continue
		}

		if list[i] == '\\' && inQuotes {
			escaped = true

			continue
		}

		if list[i] == '"' {
			inQuotes = !inQuotes
		}

		if list[i] == ',' && !inQuotes {
			tag := strings.TrimSpace(list[start:i])
			if tag != "" {
				tags = append(tags, tag)
			}

			start = i + 1
		}
	}

	tag := strings.TrimSpace(list[start:])

	if tag != "" {
		tags = append(tags, tag)
	}

	return tags
}

// etagInList reports whether etag appears in a comma-separated list using
// the weak comparison function (RFC 7232 §2.3.2): the weakness indicator
// (W/ prefix) is ignored when comparing opaque-tags.
func etagInList(list, etag string) bool {
	target := stripWeakPrefix(etag)

	for _, tag := range parseETagList(list) {
		if stripWeakPrefix(tag) == target {
			return true
		}
	}

	return false
}

// stripWeakPrefix removes the optional W/ weakness indicator from an
// entity-tag, leaving the quoted opaque-tag for comparison.
func stripWeakPrefix(etag string) string {
	return strings.TrimPrefix(etag, "W/")
}

func (w *etagWriter) isCacheableStatus() bool {
	return w.status == 0 || (w.status >= http.StatusOK && w.status < http.StatusMultipleChoices)
}

func (w *etagWriter) Flush() {
	if w.flushed {
		w.responseWrapper.Flush()

		return
	}

	w.flushed = true

	w.writeHeaderToUnderlying()

	if len(w.body) > 0 {
		_, _ = w.ResponseWriter.Write(w.body)
		w.body = w.body[:0]
	}

	w.responseWrapper.Flush()
}

func (w *etagWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.flushed = true

	return w.responseWrapper.Hijack()
}
