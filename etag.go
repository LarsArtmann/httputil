package httputil

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
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
	crc32ByteSize            = 4
	crc32HexSize             = 8
	etagWeakLen              = 12
	etagStrongLen            = 10
	defaultETagMaxBufferSize = 1024 * 1024 // 1 MB
)

//nolint:gochecknoglobals // Immutable lookup table for hex digit encoding.
var hexDigits = [16]byte{
	'0',
	'1',
	'2',
	'3',
	'4',
	'5',
	'6',
	'7',
	'8',
	'9',
	'a',
	'b',
	'c',
	'd',
	'e',
	'f',
}

// ETagConfig holds configuration for ETag generation.
type ETagConfig struct {
	Weak          bool
	MaxBufferSize int
}

// DefaultETagConfig returns an ETagConfig with sensible defaults.
func DefaultETagConfig() ETagConfig {
	return ETagConfig{
		Weak:          false,
		MaxBufferSize: defaultETagMaxBufferSize,
	}
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
}

func newETagWriter(resp http.ResponseWriter, cfg ETagConfig) *etagWriter {
	return &etagWriter{
		responseWrapper: newResponseWrapper(resp),
		body:            nil,
		weak:            cfg.Weak,
		flushed:         false,
		maxBufferSize:   cfg.MaxBufferSize,
	}
}

func (w *etagWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

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

	checksum := crc32.ChecksumIEEE(w.body)

	var buf [crc32ByteSize]byte
	binary.BigEndian.PutUint32(buf[:], checksum)

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
		dst[i*2] = hexDigits[b>>4]
		dst[i*2+1] = hexDigits[b&0x0f]
	}
}

func (w *etagWriter) matchesIfNoneMatch(req *http.Request, etag string) bool {
	inm := req.Header.Get(headerIfNoneMatch)
	if inm == "*" {
		return true
	}

	return etagInList(inm, etag)
}

func etagInList(list, etag string) bool {
	for {
		idx := strings.Index(list, ",")
		if idx < 0 {
			return strings.TrimSpace(list) == etag
		}

		if strings.TrimSpace(list[:idx]) == etag {
			return true
		}

		list = list[idx+1:]
	}
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
		//nolint:gosec // w.body is response body content (not user-influenced
		// in an XSS context); G705 taint analysis is a false positive here.
		_, _ = w.ResponseWriter.Write(w.body)
		w.body = w.body[:0]
	}

	w.responseWrapper.Flush()
}

func (w *etagWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.flushed = true

	return w.responseWrapper.Hijack()
}
