package httputil

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	defaultCompressionMinSize = 512
	headerAcceptEncoding      = "Accept-Encoding"
	headerContentEncoding     = "Content-Encoding"
	headerContentLength       = "Content-Length"
	headerVary                = "Vary"
)

const encodingGzip = "gzip"

// CompressionConfig holds configuration for response compression.
type CompressionConfig struct {
	MinSize int
	Level   int
}

// DefaultCompressionConfig returns a CompressionConfig with sensible defaults.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		MinSize: defaultCompressionMinSize,
		Level:   gzip.DefaultCompression,
	}
}

var (
	errInvalidCompressionLevel = errors.New(
		"compression level must be between gzip.HuffmanOnly and gzip.BestCompression",
	)
	errNegativeMinSize = errors.New("compression minimum size must not be negative")
)

// Validate checks the CompressionConfig for invalid values.
func (c CompressionConfig) Validate() error {
	if c.Level != gzip.DefaultCompression && (c.Level < gzip.HuffmanOnly || c.Level > gzip.BestCompression) {
		return fmt.Errorf("%w: got %d", errInvalidCompressionLevel, c.Level)
	}

	if c.MinSize < 0 {
		return fmt.Errorf("%w: got %d", errNegativeMinSize, c.MinSize)
	}

	return nil
}

// Compression returns middleware that compresses responses with gzip when the
// client accepts it and the response body exceeds the configured minimum size.
func Compression(cfg CompressionConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if !acceptsGzip(req) {
				next.ServeHTTP(resp, req)

				return
			}

			resp.Header().Add(headerVary, headerAcceptEncoding)

			cw := newCompressWriter(resp, cfg.MinSize, cfg.Level)
			defer func() { _ = cw.Close() }()

			next.ServeHTTP(cw, req)
		})
	}
}

func acceptsGzip(req *http.Request) bool {
	return strings.Contains(req.Header.Get(headerAcceptEncoding), encodingGzip)
}

type compressWriter struct {
	http.ResponseWriter

	minSize       int
	level         int
	buf           []byte
	status        int
	wroteHeader   bool
	compressing   bool
	plain         bool
	gzipWriter    *gzip.Writer
	headerWritten bool
}

func newCompressWriter(resp http.ResponseWriter, minSize, level int) *compressWriter {
	return &compressWriter{
		ResponseWriter: resp,
		minSize:        minSize,
		level:          level,
		buf:            nil,
		status:         0,
		wroteHeader:    false,
		compressing:    false,
		plain:          false,
		gzipWriter:     nil,
		headerWritten:  false,
	}
}

func (w *compressWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
}

func (w *compressWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.plain {
		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, fmt.Errorf("failed to write to response writer: %w", err)
		}

		return n, nil
	}

	if w.compressing {
		n, err := w.gzipWriter.Write(b)
		if err != nil {
			return n, errorfamily.WrapTransient(err, ErrCodeCompressWriteFailed, "gzip writer write failed")
		}

		return n, nil
	}

	w.buf = append(w.buf, b...)

	if len(w.buf) >= w.minSize && w.shouldCompress() {
		err := w.startCompression()
		if err != nil {
			return 0, fmt.Errorf("failed to start gzip compression: %w", err)
		}

		return len(b), nil
	}

	return len(b), nil
}

func (w *compressWriter) shouldCompress() bool {
	if w.status < http.StatusOK || w.status >= http.StatusMultipleChoices {
		return false
	}

	if w.Header().Get(headerContentEncoding) != "" {
		return false
	}

	return true
}

func (w *compressWriter) startCompression() error {
	w.Header().Set(headerContentEncoding, encodingGzip)
	w.Header().Del(headerContentLength)

	gz, err := gzip.NewWriterLevel(w.ResponseWriter, w.level)
	if err != nil {
		return errorfamily.WrapTransient(err, ErrCodeCompressWriteFailed, "gzip writer creation failed")
	}

	w.gzipWriter = gz
	w.compressing = true

	if w.wroteHeader && !w.headerWritten {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerWritten = true
	}

	if len(w.buf) > 0 {
		_, err := w.gzipWriter.Write(w.buf)
		w.buf = w.buf[:0]

		if err != nil {
			return errorfamily.WrapTransient(err, ErrCodeCompressWriteFailed, "gzip writer buffered write failed")
		}
	}

	return nil
}

func (w *compressWriter) Close() error {
	if w.plain {
		return nil
	}

	if w.compressing {
		err := w.gzipWriter.Close()
		if err != nil {
			return errorfamily.WrapTransient(err, ErrCodeCompressWriteFailed, "gzip writer close failed")
		}

		return nil
	}

	if w.wroteHeader && !w.headerWritten {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerWritten = true
	}

	if len(w.buf) > 0 {
		_, err := w.ResponseWriter.Write(w.buf)
		if err != nil {
			return errorfamily.WrapTransient(err, ErrCodeCompressWriteFailed, "buffered response write failed")
		}
	}

	return nil
}

func (w *compressWriter) Flush() {
	if w.compressing {
		_ = w.gzipWriter.Flush()

		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}

		return
	}

	if w.plain {
		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}

		return
	}

	w.plain = true

	if w.wroteHeader && !w.headerWritten {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerWritten = true
	}

	if len(w.buf) > 0 {
		_, _ = w.ResponseWriter.Write(w.buf)
		w.buf = w.buf[:0]
	}

	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.plain = true

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

func (w *compressWriter) Push(target string, opts *http.PushOptions) error {
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
