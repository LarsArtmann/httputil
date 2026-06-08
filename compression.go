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
	headerContentType         = "Content-Type"
	headerVary                = "Vary"
)

const encodingGzip = "gzip"

//nolint:gochecknoglobals // sync.Pool is inherently package-level for writer reuse.
var (
	gzipWriterPools   = make(map[int]*sync.Pool)
	gzipWriterPoolsMu sync.RWMutex
)

func getGzipPool(level int) *sync.Pool {
	gzipWriterPoolsMu.RLock()

	pool, ok := gzipWriterPools[level]

	gzipWriterPoolsMu.RUnlock()

	if ok {
		return pool
	}

	gzipWriterPoolsMu.Lock()
	defer gzipWriterPoolsMu.Unlock()

	pool, ok = gzipWriterPools[level]
	if ok {
		return pool
	}

	pool = &sync.Pool{
		New: func() any {
			gz, err := gzip.NewWriterLevel(io.Discard, level)
			if err != nil {
				panic("gzip.NewWriterLevel(" + itoa(level) + "): " + err.Error())
			}

			return gz
		},
	}
	gzipWriterPools[level] = pool

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
	if c.Level != gzip.DefaultCompression &&
		(c.Level < gzip.HuffmanOnly || c.Level > gzip.BestCompression) {
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

	switch {
	case w.plain:
		return w.writePlain(b)
	case w.compressing:
		return w.writeCompressed(b)
	default:
		return w.writeBuffered(b)
	}
}

func (w *compressWriter) writePlain(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("failed to write to response writer: %w", err)
	}

	return n, nil
}

func (w *compressWriter) writeCompressed(b []byte) (int, error) {
	n, err := w.gzipWriter.Write(b)
	if err != nil {
		return n, errorfamily.WrapTransient(
			err,
			ErrCodeCompressWriteFailed,
			"gzip writer write failed",
		)
	}

	return n, nil
}

func (w *compressWriter) writeBuffered(b []byte) (int, error) {
	total := len(b)

	// Buffer only up to minSize so large responses do not accumulate
	// indefinitely before the compress-or-not decision is made.
	needed := w.minSize - len(w.buf)
	if needed > 0 {
		if len(b) <= needed {
			w.buf = append(w.buf, b...)

			if len(w.buf) < w.minSize {
				return total, nil
			}
		} else {
			w.buf = append(w.buf, b[:needed]...)
			b = b[needed:]
		}
	}

	if w.shouldCompress() {
		return w.startCompressAndStream(b, total)
	}

	return w.flushPlainAndStream(b, total)
}

func (w *compressWriter) startCompressAndStream(b []byte, total int) (int, error) {
	err := w.startCompression()
	if err != nil {
		return 0, fmt.Errorf("failed to start gzip compression: %w", err)
	}

	if len(b) == 0 {
		return total, nil
	}

	_, err = w.gzipWriter.Write(b)
	if err != nil {
		return total, errorfamily.WrapTransient(
			err, ErrCodeCompressWriteFailed, "gzip writer streaming write failed",
		)
	}

	return total, nil
}

func (w *compressWriter) flushPlainAndStream(b []byte, total int) (int, error) {
	w.plain = true
	w.writeHeaderToUnderlying()

	if len(w.buf) > 0 {
		_, err := w.ResponseWriter.Write(w.buf)
		if err != nil {
			return 0, fmt.Errorf("failed to write buffered plain response: %w", err)
		}

		w.buf = w.buf[:0]
	}

	if len(b) == 0 {
		return total, nil
	}

	_, err := w.ResponseWriter.Write(b)
	if err != nil {
		return total, fmt.Errorf("failed to write plain response: %w", err)
	}

	return total, nil
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

	if !isCompressibleContentType(w.Header().Get(headerContentType)) {
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
			return errorfamily.WrapTransient(
				err,
				ErrCodeCompressWriteFailed,
				"gzip writer buffered write failed",
			)
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
			return errorfamily.WrapTransient(
				err,
				ErrCodeCompressWriteFailed,
				"gzip writer close failed",
			)
		}

		getGzipPool(w.level).Put(w.gzipWriter)
		w.gzipWriter = nil

		return nil
	}

	w.writeHeaderToUnderlying()

	if len(w.buf) > 0 {
		_, err := w.ResponseWriter.Write(w.buf)
		if err != nil {
			return errorfamily.WrapTransient(
				err,
				ErrCodeCompressWriteFailed,
				"buffered response write failed",
			)
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
