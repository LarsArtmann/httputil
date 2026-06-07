package httputil

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
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
	defaultETagMaxBufferSize = 1024 * 1024 // 1 MB
)

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
	http.ResponseWriter

	body          []byte
	status        int
	wroteHeader   bool
	weak          bool
	flushed       bool
	headerWritten bool
	maxBufferSize int
}

func newETagWriter(resp http.ResponseWriter, cfg ETagConfig) *etagWriter {
	return &etagWriter{
		ResponseWriter: resp,
		body:           nil,
		status:         0,
		wroteHeader:    false,
		weak:           cfg.Weak,
		flushed:        false,
		headerWritten:  false,
		maxBufferSize:  cfg.MaxBufferSize,
	}
}

func (w *etagWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
}

func (w *etagWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.flushed {
		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, errorfamily.WrapTransient(err, ErrCodeETagWriteFailed, "etag writer streaming write failed")
		}

		return n, nil
	}

	if w.maxBufferSize > 0 && len(w.body)+len(b) > w.maxBufferSize {
		w.Flush()

		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, errorfamily.WrapTransient(err, ErrCodeETagWriteFailed, "etag writer overflow write failed")
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
			if w.wroteHeader && !w.headerWritten {
				w.ResponseWriter.WriteHeader(http.StatusNotModified)
				w.headerWritten = true
			}

			return
		}
	}

	if w.wroteHeader && !w.headerWritten {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerWritten = true
	}

	if len(w.body) > 0 {
		_, _ = w.ResponseWriter.Write(w.body)
	}
}

func (w *etagWriter) computeETag() string {
	if len(w.body) == 0 && !w.wroteHeader {
		return ""
	}

	checksum := crc32.ChecksumIEEE(w.body)
	buf := make([]byte, crc32ByteSize)
	binary.BigEndian.PutUint32(buf, checksum)
	hexStr := hex.EncodeToString(buf)

	if w.weak {
		return `W/"` + hexStr + `"`
	}

	return `"` + hexStr + `"`
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
		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}

		return
	}

	w.flushed = true

	if w.wroteHeader && !w.headerWritten {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerWritten = true
	}

	if len(w.body) > 0 {
		_, _ = w.ResponseWriter.Write(w.body)
		w.body = w.body[:0]
	}

	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *etagWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.flushed = true

	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errorfamily.WrapInfrastructure(
			http.ErrNotSupported, ErrCodeHijackUnsupported, "response writer does not implement http.Hijacker",
		)
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return conn, rw, errorfamily.WrapTransient(err, ErrCodeHijackFailed, "response writer hijack failed")
	}

	return conn, rw, nil
}

func (w *etagWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return errorfamily.WrapInfrastructure(
			http.ErrNotSupported, ErrCodePushUnsupported, "response writer does not implement http.Pusher",
		).WithContext("target", target)
	}

	err := pusher.Push(target, opts)
	if err != nil {
		return errorfamily.WrapTransient(err, ErrCodePushFailed, "response writer push failed").
			WithContext("target", target)
	}

	return nil
}
