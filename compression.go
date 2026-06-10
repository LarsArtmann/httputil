package httputil

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
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
	errNegativeMinSize = errors.New("compression minimum size must not be negative")
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
