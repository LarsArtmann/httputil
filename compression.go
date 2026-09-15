package httputil

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

const (
	defaultCompressionMinSize = 512
)

const (
	encodingGzip     = "gzip"
	encodingDeflate  = "deflate"
	encodingIdentity = "identity"
	encodingBr       = "br"
	encodingZstd     = "zstd"
)

const defaultQValue = 1.0

// WriterFactory creates a fresh streaming writer that compresses bytes written
// to it and writes them to dst. Implementations are responsible for buffering
// and finalization. The returned writer must be Close()d by the caller to flush
// pending compressed bytes.
//
// The factory is called once per response. Implementations may be expensive
// to construct; consider pooling if hot.
type WriterFactory func(dst io.Writer) (io.WriteCloser, error)

// GzipWriterFactory returns a WriterFactory backed by compress/gzip at the
// given level (gzip.DefaultCompression, gzip.BestSpeed, etc.).
func GzipWriterFactory(level int) WriterFactory {
	return func(dst io.Writer) (io.WriteCloser, error) {
		return gzip.NewWriterLevel(dst, level)
	}
}

// AbsentEncodingPolicy selects the response representation for requests that
// carry no Accept-Encoding header (or an empty one). RFC 7231 §5.3.4: an
// absent header signals no explicit client preference, and an empty value
// signals that no content-coding is wanted.
type AbsentEncodingPolicy int

const (
	// AbsentEncodingIdentity (zero value) serves the uncompressed
	// representation to requests without an Accept-Encoding header. Minimal
	// HTTP clients, health probes, and intermediaries that do not declare
	// support may not decompress a compressed response, so not compressing
	// is the safe default.
	AbsentEncodingIdentity AbsentEncodingPolicy = iota

	// AbsentEncodingFirstConfigured restores the v1.1.x behavior: requests
	// without an Accept-Encoding header receive the highest-priority
	// configured encoding. Use it only for deployments whose clients are
	// known to decompress every response.
	AbsentEncodingFirstConfigured
)

// CompressionConfig holds configuration for response compression.
//
// The default config negotiates between gzip and deflate based on the client's
// Accept-Encoding header (q-values respected), compresses payloads >= 512 B,
// uses gzip's default level, and skips incompressible content types
// (images, video, audio, and pre-compressed application/* types).
//
// To support encodings not bundled with this package (brotli, zstd, lz4),
// supply WriterFactory entries in WriterFactories. Encoding negotiation will
// include any registered encoding whose name appears in the Accept-Encoding
// header.
type CompressionConfig struct {
	// MinSize is the minimum response body size (in bytes) before compression
	// is attempted. Responses smaller than this are sent uncompressed.
	MinSize int

	// Level is the compression level passed to default factories when
	// WriterFactories is not supplied. Ignored when WriterFactories is set.
	// Valid range: gzip.HuffmanOnly to gzip.BestCompression, or
	// gzip.DefaultCompression (-1).
	//
	// The zero value means "unset": Compression() treats Level == 0 as
	// gzip.DefaultCompression, so a zero-value config compresses at the
	// default level. A genuinely uncompressed response is not expressible
	// through Level; use the identity encoding or skip the middleware.
	Level int

	// WriterFactories maps canonical encoding names to factory functions.
	// The default set contains "gzip" and "deflate". Add entries such as
	// "br" or "zstd" to enable brotli/zstd (the caller must provide a
	// factory backed by the desired implementation).
	//
	// Built-in factory map is replaced entirely by the supplied map; copy
	// defaults via DefaultWriterFactories() to extend rather than replace.
	WriterFactories map[string]WriterFactory

	// IncompressibleTypes is a list of content-type prefixes that should
	// not be compressed (e.g., "image/", "video/"). When nil, the defaults
	// from DefaultIncompressibleTypes() are used. Set to an empty slice to
	// compress all content types.
	IncompressibleTypes []string

	// AbsentEncoding selects the response representation for requests that
	// carry no Accept-Encoding header (or an empty one).
	//
	// The zero value is AbsentEncodingIdentity: the response is sent
	// uncompressed. Set AbsentEncodingFirstConfigured to restore the v1.1.x
	// behavior of compressing with the highest-priority configured encoding.
	AbsentEncoding AbsentEncodingPolicy
}

// DefaultWriterFactories returns a fresh map containing the stdlib encodings
// (gzip, deflate, identity) at gzip.DefaultCompression. Useful when extending
// WriterFactories without dropping the built-ins.
func DefaultWriterFactories() map[string]WriterFactory {
	return DefaultWriterFactoriesForLevel(gzip.DefaultCompression)
}

// DefaultWriterFactoriesForLevel returns a fresh map containing the stdlib
// encodings (gzip, deflate, identity) at the given compression level. Use this
// when you want CompressionConfig.Level to take effect without supplying a
// custom WriterFactories map.
//
// Unlike CompressionConfig.Level, the level parameter follows compress/gzip
// semantics literally: 0 means gzip.NoCompression here, not "unset". Pass
// gzip.DefaultCompression (-1) for the default level.
func DefaultWriterFactoriesForLevel(level int) map[string]WriterFactory {
	return map[string]WriterFactory{
		encodingGzip:     GzipWriterFactory(level),
		encodingDeflate:  DeflateWriterFactory(level),
		encodingIdentity: passthroughFactory,
	}
}

// passthroughFactory returns a writer that simply forwards to dst. Used
// for the "identity" encoding to indicate "no compression".
func passthroughFactory(dst io.Writer) (io.WriteCloser, error) {
	return nopCloserWriter{dst}, nil
}

// DeflateWriterFactory returns a WriterFactory backed by compress/flate
// (raw deflate stream, no zlib header) at the given level. Browsers expecting
// "deflate" almost always tolerate raw deflate, but some older proxies
// required zlib-wrapped deflate; raw deflate is the HTTP-standard form.
func DeflateWriterFactory(level int) WriterFactory {
	return func(dst io.Writer) (io.WriteCloser, error) {
		// Use flate.NewWriter for raw deflate; -1 = default compression
		return flate.NewWriter(dst, level)
	}
}

// DefaultCompressionConfig returns a CompressionConfig with sensible defaults
// for a typical JSON/text API: negotiates gzip/deflate, compresses payloads
// >= 512 B at gzip's default level.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		MinSize:             defaultCompressionMinSize,
		Level:               gzip.DefaultCompression,
		WriterFactories:     DefaultWriterFactories(),
		IncompressibleTypes: nil,
		AbsentEncoding:      AbsentEncodingIdentity,
	}
}

// Error codes for CompressionConfig validation and Accept-Encoding q-value
// parsing. Config codes are Rejection (invalid input); q-value codes are
// Rejection too — a malformed Accept-Encoding header is the client's input.
const (
	codeCompressionLevelInvalid          = Code("compression.level_invalid")
	codeCompressionMinSizeNeg            = Code("compression.min_size_negative")
	codeCompressionNoFactory             = Code("compression.no_writer_factory")
	codeCompressionIncompressibleInvalid = Code("compression.incompressible_prefix_invalid")
	codeCompressionAbsentEncodingInvalid = Code("compression.absent_encoding_invalid")
	codeCompressionQValueEmpty           = Code("compression.qvalue_empty")
	codeCompressionQValueInvalid         = Code("compression.qvalue_invalid_int")
	codeCompressionQValueTrail           = Code("compression.qvalue_trailing_chars")
	codeCompressionQValueTooBig          = Code("compression.qvalue_too_large")
)

var (
	errInvalidCompressionLevel = codeCompressionLevelInvalid.Rejection(
		"compression level must be between gzip.HuffmanOnly and gzip.BestCompression",
	)
	errNegativeMinSize = codeCompressionMinSizeNeg.Rejection(
		"compression minimum size must not be negative",
	)
	errNoWriterFactory = codeCompressionNoFactory.Rejection(
		"compression WriterFactories is empty; at least one encoding is required",
	)
	errIncompressiblePrefixInvalid = codeCompressionIncompressibleInvalid.Rejection(
		"compression IncompressibleTypes entries must be non-empty media-type prefixes such as \"image/\"",
	)
	errAbsentEncodingInvalid = codeCompressionAbsentEncodingInvalid.Rejection(
		"compression AbsentEncoding policy is not a known AbsentEncodingPolicy value",
	)
	errEmptyQValue    = codeCompressionQValueEmpty.Rejection("empty q-value")
	errInvalidQInt    = codeCompressionQValueInvalid.Rejection("invalid q-value integer")
	errTrailingQChars = codeCompressionQValueTrail.Rejection("trailing chars in q-value")
	errQValueTooLarge = codeCompressionQValueTooBig.Rejection("q-value > 1.0")
)

// Validate checks the CompressionConfig for invalid values. Level == 0 is
// accepted as the zero-value shorthand for gzip.DefaultCompression (see
// CompressionConfig.Level). The Level range check applies only when
// WriterFactories is empty — that is the only case where Level drives
// factory construction; with explicit factories, Level is ignored and is
// not validated, so configs that set both do not produce spurious warnings.
func (c CompressionConfig) Validate() error {
	if len(c.WriterFactories) == 0 &&
		c.Level != gzip.DefaultCompression &&
		(c.Level < gzip.HuffmanOnly || c.Level > gzip.BestCompression) {
		return errInvalidCompressionLevel.WithContextAny("level", c.Level)
	}

	if c.MinSize < 0 {
		return errNegativeMinSize.WithContextAny("min_size", c.MinSize)
	}

	if err := c.validateAbsentEncoding(); err != nil {
		return err
	}

	if len(c.WriterFactories) == 0 {
		return errNoWriterFactory
	}

	for _, prefix := range c.IncompressibleTypes {
		if prefix == "" || strings.TrimSpace(prefix) != prefix || !strings.Contains(prefix, "/") {
			return errIncompressiblePrefixInvalid.WithContext("prefix", prefix)
		}
	}

	return nil
}

// validateAbsentEncoding rejects AbsentEncoding values that are not one of
// the defined AbsentEncodingPolicy constants.
func (c CompressionConfig) validateAbsentEncoding() error {
	known := c.AbsentEncoding == AbsentEncodingIdentity ||
		c.AbsentEncoding == AbsentEncodingFirstConfigured
	if !known {
		return errAbsentEncodingInvalid.WithContextAny("policy", int(c.AbsentEncoding))
	}

	return nil
}

// Compression returns middleware that compresses responses based on the
// client's Accept-Encoding header and the configured factory map. The
// negotiation respects RFC 7231 q-values.
//
// If no compatible encoding is found, the response is sent uncompressed.
func Compression(cfg CompressionConfig) Middleware {
	// Fall back to defaults if the caller didn't supply a factory map.
	// This preserves backward compatibility with configs created before
	// the WriterFactories field existed.
	//
	// Level == 0 means "unset" (see CompressionConfig.Level), so the
	// zero-value config compresses at gzip.DefaultCompression. Because the
	// level becomes the factories' level here, it is range-checked at this
	// use site; Validate skips it once WriterFactories is filled.
	if len(cfg.WriterFactories) == 0 {
		level := cfg.Level
		if level == 0 {
			level = gzip.DefaultCompression
		}

		if level != gzip.DefaultCompression &&
			(level < gzip.HuffmanOnly || level > gzip.BestCompression) {
			validateConfig(
				"CompressionConfig",
				errInvalidCompressionLevel.WithContextAny("level", cfg.Level),
			)
		}

		cfg.WriterFactories = DefaultWriterFactoriesForLevel(level)
	}

	validateConfig("CompressionConfig", cfg.Validate())

	neg := buildNegotiator(cfg.WriterFactories, cfg.AbsentEncoding)

	skipTypes := cfg.IncompressibleTypes
	if skipTypes == nil {
		skipTypes = DefaultIncompressibleTypes()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			encoding, _, ok := neg.negotiateEncoding(req.Header.Get(headerAcceptEncoding))
			if !ok || encoding == encodingIdentity {
				// No compression possible or client wants identity.
				resp.Header().Add(headerVary, headerAcceptEncoding)

				next.ServeHTTP(resp, req)

				return
			}

			resp.Header().Add(headerVary, headerAcceptEncoding)

			factory := cfg.WriterFactories[encoding]

			writer := newCompressWriter(
				resp,
				cfg.MinSize,
				encoding,
				factory,
				neg.poolFor(encoding),
				skipTypes,
			)

			// Cleanup: handler has returned and response is in-flight.
			defer func() { _ = writer.Close() }()

			next.ServeHTTP(writer, req)
		})
	}
}
