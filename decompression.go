package httputil

import (
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"
)

const defaultMaxDecompressionSize = 1 << 24 // 16 MiB

// Error codes for DecompressionConfig validation (Rejection: invalid input)
// and runtime decompression failures.
const (
	codeDecompressionSizeNegative = Code("decompression.max_size_negative")
	codeDecompressionSizeExceeded = Code("decompression.size_exceeded")
	codeDecompressionReadFailed   = Code("decompression.read_failed")
	codeDecompressionCloseFailed  = Code("decompression.close_failed")
)

var errMaxDecompressionSizeNegative = codeDecompressionSizeNegative.Rejection(
	"DecompressionConfig.MaxDecompressionSize must not be negative",
)

// errDecompressionSizeExceeded is the bomb-protection trip: the decompressed
// body exceeded the configured limit. Classified as Rejection — the client
// sent a decompression bomb, retrying the same request cannot succeed.
var errDecompressionSizeExceeded = codeDecompressionSizeExceeded.Rejection(
	"decompression size limit exceeded",
)

// DecompressionConfig holds the configuration for the Decompression middleware.
type DecompressionConfig struct {
	// Encodings specifies which request body encodings to decompress.
	// Supported values: "gzip", "deflate". Empty = both (default).
	Encodings []string

	// MaxDecompressionSize limits the decompressed body size in bytes to
	// prevent decompression bombs (zip bombs). Zero means no limit.
	// Default: 16 MiB.
	MaxDecompressionSize int64
}

// DefaultDecompressionConfig returns a DecompressionConfig that decompresses
// both gzip and deflate with a 16 MiB decompression size limit.
func DefaultDecompressionConfig() DecompressionConfig {
	return DecompressionConfig{
		Encodings:            []string{encodingGzip, encodingDeflate},
		MaxDecompressionSize: defaultMaxDecompressionSize,
	}
}

// Validate checks the DecompressionConfig for invalid values.
func (c DecompressionConfig) Validate() error {
	if c.MaxDecompressionSize < 0 {
		return errMaxDecompressionSizeNegative.WithContextAny(
			"max_decompression_size",
			c.MaxDecompressionSize,
		)
	}

	return nil
}

// Decompression returns middleware that decompresses request bodies based on
// the Content-Encoding header. Supported encodings: gzip, deflate.
//
// The middleware wraps r.Body with the appropriate decompressor and removes
// the Content-Encoding and Content-Length headers from the request so downstream
// handlers see the decompressed body transparently.
//
// To prevent decompression bombs, the decompressed body is limited to
// DecompressionConfig.MaxDecompressionSize bytes (default: 16 MiB). A
// decompression bomb attack sends a small compressed payload that decompresses
// to an enormous size, exhausting server memory.
func Decompression(cfg DecompressionConfig) Middleware {
	validateConfig("DecompressionConfig", cfg.Validate())

	allowed := make(map[string]bool)
	for _, enc := range cfg.Encodings {
		allowed[strings.ToLower(strings.TrimSpace(enc))] = true
	}

	if len(allowed) == 0 {
		allowed[encodingGzip] = true
		allowed[encodingDeflate] = true
	}

	maxSize := cfg.MaxDecompressionSize
	if maxSize == 0 {
		maxSize = defaultMaxDecompressionSize
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil {
				next.ServeHTTP(w, r)

				return
			}

			encoding := r.Header.Get(headerContentEncoding)
			if encoding == "" || !allowed[strings.ToLower(encoding)] {
				next.ServeHTTP(w, r)

				return
			}

			var reader io.ReadCloser

			switch strings.ToLower(encoding) {
			case encodingGzip:
				gzipReader, err := gzip.NewReader(r.Body)
				if err != nil {
					http.Error(w, "invalid gzip body", http.StatusBadRequest)

					return
				}

				reader = gzipReader
			case encodingDeflate:
				reader = flate.NewReader(r.Body)
			default:
				next.ServeHTTP(w, r)

				return
			}

			r.Body = limitedReadCloser(reader, maxSize)
			r.Header.Del(headerContentEncoding)
			r.Header.Del(headerContentLength)

			next.ServeHTTP(w, r)
		})
	}
}

// limitedReadCloser wraps a ReadCloser with a maximum byte limit.
// If the limit is exceeded, reads return an error.
func limitedReadCloser(rc io.ReadCloser, limit int64) io.ReadCloser {
	return &limitedReader{rc: rc, limit: limit, remaining: limit}
}

type limitedReader struct {
	rc        io.ReadCloser
	limit     int64
	remaining int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		// Bomb-protection cleanup; error already returned to caller.
		_ = l.rc.Close()

		return 0, errDecompressionSizeExceeded
	}

	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}

	n, err := l.rc.Read(p)
	l.remaining -= int64(n)

	if err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}

		return n, codeDecompressionReadFailed.WrapCorruption(err, "decompression read failed")
	}

	return n, nil
}

func (l *limitedReader) Close() error {
	err := l.rc.Close()
	if err != nil {
		return codeDecompressionCloseFailed.WrapTransient(err, "decompression close failed")
	}

	return nil
}
