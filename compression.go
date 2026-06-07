package httputil

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

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

//nolint:gochecknoglobals // sync.Pool is inherently package-level for writer reuse.
var gzipWriterPools = make(map[int]*sync.Pool)

func getGzipPool(level int) *sync.Pool {
	pool, ok := gzipWriterPools[level]
	if !ok {
		pool = &sync.Pool{
			New: func() any {
				gz, _ := gzip.NewWriterLevel(io.Discard, level)

				return gz
			},
		}
		gzipWriterPools[level] = pool
	}

	return pool
}

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
	errNegativeMinSize  = errors.New("compression minimum size must not be negative")
	errPoolTypeMismatch = errors.New("unexpected type from gzip writer pool")
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
	responseWrapper

	minSize     int
	level       int
	buf         []byte
	compressing bool
	plain       bool
	gzipWriter  *gzip.Writer
}

func newCompressWriter(resp http.ResponseWriter, minSize, level int) *compressWriter {
	return &compressWriter{
		responseWrapper: newResponseWrapper(resp),
		minSize:         minSize,
		level:           level,
		buf:             nil,
		compressing:     false,
		plain:           false,
		gzipWriter:      nil,
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

func (w *compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.plain = true

	return w.responseWrapper.Hijack()
}

func (w *compressWriter) shouldCompress() bool {
	if w.status < http.StatusOK || w.status >= http.StatusMultipleChoices {
		return false
	}

	if w.Header().Get(headerContentEncoding) != "" {
		return false
	}

	if !isCompressibleContentType(w.Header().Get("Content-Type")) {
		return false
	}

	return true
}

//nolint:gochecknoglobals // Immutable reference data for content-type filtering.
var incompressiblePrefixes = []string{
	"image/",
	"video/",
	"audio/",
	"application/gzip",
	"application/zip",
	"application/pdf",
	"application/x-rar",
	"application/x-7z",
	"application/x-compress",
}

func isCompressibleContentType(contentType string) bool {
	if contentType == "" {
		return true
	}

	for _, prefix := range incompressiblePrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return false
		}
	}

	return true
}

func (w *compressWriter) startCompression() error {
	w.Header().Set(headerContentEncoding, encodingGzip)
	w.Header().Del(headerContentLength)

	pool := getGzipPool(w.level)
	raw := pool.Get()

	gzipWriter, ok := raw.(*gzip.Writer)
	if !ok {
		return errorfamily.WrapTransient(
			errPoolTypeMismatch,
			ErrCodeCompressWriteFailed,
			"gzip writer pool type mismatch",
		)
	}

	gzipWriter.Reset(w.ResponseWriter)

	w.gzipWriter = gzipWriter
	w.compressing = true

	w.writeHeaderToUnderlying()

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

		getGzipPool(w.level).Put(w.gzipWriter)
		w.gzipWriter = nil

		return nil
	}

	w.writeHeaderToUnderlying()

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

		w.responseWrapper.Flush()

		return
	}

	if w.plain {
		w.responseWrapper.Flush()

		return
	}

	w.plain = true

	w.writeHeaderToUnderlying()

	if len(w.buf) > 0 {
		_, _ = w.ResponseWriter.Write(w.buf)
		w.buf = w.buf[:0]
	}

	w.responseWrapper.Flush()
}
