package httputil

import (
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultCompressionMinSize = 512
	headerAcceptEncoding      = "Accept-Encoding"
	headerContentEncoding     = "Content-Encoding"
	headerContentLength       = "Content-Length"
	headerContentType         = "Content-Type"
	headerVary                = "Vary"
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

	// Level is the gzip compression level (when gzip is selected). Ignored
	// for other encodings unless their factory consults cfg.Level via closure.
	Level int

	// WriterFactories maps canonical encoding names to factory functions.
	// The default set contains "gzip" and "deflate". Add entries such as
	// "br" or "zstd" to enable brotli/zstd (the caller must provide a
	// factory backed by the desired implementation).
	//
	// Built-in factory map is replaced entirely by the supplied map; copy
	// defaults via DefaultWriterFactories() to extend rather than replace.
	WriterFactories map[string]WriterFactory

	// QValues maps encoding name to its quality value override (0.0-1.0).
	// An encoding with q=0 in the client header is excluded from selection.
	// This map does not affect parsing of client q-values; it only lets
	// the server hint at preference for clients that omit a quality value.
	QValues map[string]float64
}

// DefaultWriterFactories returns a fresh map containing the stdlib encodings
// (gzip, deflate, identity). Useful when extending WriterFactories without
// dropping the built-ins.
func DefaultWriterFactories() map[string]WriterFactory {
	return map[string]WriterFactory{
		encodingGzip:     GzipWriterFactory(gzip.DefaultCompression),
		encodingDeflate:  DeflateWriterFactory(gzip.DefaultCompression),
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
		MinSize:         defaultCompressionMinSize,
		Level:           gzip.DefaultCompression,
		WriterFactories: DefaultWriterFactories(),
		QValues:         nil,
	}
}

var (
	errInvalidCompressionLevel = errors.New(
		"compression level must be between gzip.HuffmanOnly and gzip.BestCompression",
	)
	errNegativeMinSize = errors.New("compression minimum size must not be negative")
	errNoWriterFactory = errors.New(
		"compression WriterFactories is empty; at least one encoding is required",
	)
	errEmptyQValue    = errors.New("empty q-value")
	errInvalidQInt    = errors.New("invalid q-value integer")
	errTrailingQChars = errors.New("trailing chars in q-value")
	errQValueTooLarge = errors.New("q-value > 1.0")
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

	if len(c.WriterFactories) == 0 {
		return errNoWriterFactory
	}

	return nil
}

// negotiator pre-compiles encoding priority at config time so per-request
// parsing only does simple string matching.
type negotiator struct {
	// Priority order of encodings, highest quality first.
	// Each entry is the canonical name (lowercased).
	order []string
	// factories maps encoding name to its factory.
	factories map[string]WriterFactory
}

// buildNegotiator pre-parses the factory map and assigns each encoding a
// stable priority index. The negotiator's order field is the priority list
// used for tiebreaking when two encodings have identical q-values.
func buildNegotiator(factories map[string]WriterFactory) *negotiator {
	priorityOf := func(name string) int {
		for i, preferred := range preferredEncodingOrder {
			if preferred == name {
				return i
			}
		}

		// Unknown encoding: sort after all built-ins, alphabetically.
		return len(preferredEncodingOrder) + nameOffset(name)
	}

	order := make([]string, 0, len(factories))
	priorities := make([]int, 0, len(factories))

	for name := range factories {
		order = append(order, name)
		priorities = append(priorities, priorityOf(name))
	}

	// Insertion sort by priority. n is tiny (1-5 encodings).
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && priorities[j] < priorities[j-1]; j-- {
			order[j-1], order[j] = order[j], order[j-1]
			priorities[j-1], priorities[j] = priorities[j], priorities[j-1]
		}
	}

	return &negotiator{
		order:     order,
		factories: factories,
	}
}

// nameOffset returns a stable hash of name's bytes for ordering unknown
// encodings alphabetically without importing "sort".
func nameOffset(name string) int {
	const byteBase = 256

	offset := 0
	for i := range len(name) {
		offset = offset*byteBase + int(name[i])
	}

	return offset
}

// preferredEncodingOrder is the canonical server-side preference when two
// encodings tie on client q-value. Matches what major browsers and CDNs
// negotiate: brotli (smallest) > zstd > gzip > deflate > identity.
//
//nolint:gochecknoglobals // Immutable lookup table for encoding preference.
var preferredEncodingOrder = []string{
	encodingBr,
	encodingZstd,
	encodingGzip,
	encodingDeflate,
	encodingIdentity,
}

// negotiateEncoding parses an Accept-Encoding header and returns the
// highest-priority encoding available in the negotiator. The returned
// encoding name and q-value reflect the client request; the factory
// comes from the negotiator's pre-built map.
//
// Returns ("", 0, false) if no acceptable encoding is found (client
// excluded every available encoding via q=0 or sent *; q=0).
func (n *negotiator) negotiateEncoding(header string) (string, float64, bool) {
	if header == "" {
		return n.negotiateEmptyHeader()
	}

	bestName, bestQ := n.scanAcceptEncoding(header)
	if bestName == "" {
		return n.fallbackToIdentity()
	}

	return bestName, bestQ, true
}

// negotiateEmptyHeader handles the "no Accept-Encoding header" case: pick
// the first configured encoding (deterministic via sorted order).
func (n *negotiator) negotiateEmptyHeader() (string, float64, bool) {
	if len(n.order) > 0 {
		return n.order[0], defaultQValue, true
	}

	return "", 0, false
}

// scanAcceptEncoding walks the comma-separated entries in header and
// returns the best-supported encoding (by q-value, then by server order)
// along with its q-value. Returns ("", 0) if no supported encoding is found.
func (n *negotiator) scanAcceptEncoding(header string) (string, float64) {
	bestName := ""
	bestQ := -1.0
	bestOrder := len(n.order) + 1

	pos := 0

	for pos < len(header) {
		pos = skipEntrySeparators(header, pos)
		start := pos

		pos = findEntryEnd(header, pos)

		entry := header[start:trimRightWhitespace(header, start, pos)]
		if entry == "" {
			continue
		}

		name, quality := parseEncodingEntry(entry)
		if quality <= 0 {
			// q=0 explicitly disables this encoding.
			continue
		}

		orderIdx := indexOf(n.order, name)
		if orderIdx < 0 {
			continue
		}

		if quality > bestQ || (quality == bestQ && orderIdx < bestOrder) {
			bestName = name
			bestQ = quality
			bestOrder = orderIdx
		}
	}

	return bestName, bestQ
}

// fallbackToIdentity returns the identity encoding if registered, else
// signals failure. Used when the client sent a header that excluded
// every compression encoding (e.g., q=0 on all of them).
func (n *negotiator) fallbackToIdentity() (string, float64, bool) {
	if _, ok := n.factories[encodingIdentity]; ok {
		return encodingIdentity, defaultQValue, true
	}

	return "", 0, false
}

// skipEntrySeparators advances past leading whitespace and commas.
func skipEntrySeparators(header string, pos int) int {
	for pos < len(header) && isEntrySeparator(header[pos]) {
		pos++
	}

	return pos
}

// isEntrySeparator reports whether b is a whitespace or comma character
// that separates Accept-Encoding entries.
func isEntrySeparator(b byte) bool {
	return b == ' ' || b == '\t' || b == ','
}

// findEntryEnd advances pos to the next comma or end of header.
func findEntryEnd(header string, pos int) int {
	for pos < len(header) && header[pos] != ',' {
		pos++
	}

	return pos
}

// trimRightWhitespace returns the end position of header[start:end]
// with trailing whitespace removed.
func trimRightWhitespace(header string, start, end int) int {
	for end > start && isEntrySeparator(header[end-1]) && header[end-1] != ',' {
		end--
	}

	return end
}

// parseEncodingEntry parses a single entry like "gzip" or "gzip;q=0.8"
// and returns the canonical encoding name and q-value.
func parseEncodingEntry(entry string) (string, float64) {
	semi := indexByte(entry, ';')
	if semi < 0 {
		return strings.ToLower(trim(entry)), defaultQValue
	}

	name := strings.ToLower(trim(entry[:semi]))
	rest := trim(entry[semi+1:])

	if !strings.HasPrefix(rest, qValuePrefix) {
		return name, defaultQValue
	}

	qStr := rest[len(qValuePrefix):]

	q, err := parseQValue(qStr)
	if err != nil {
		return name, defaultQValue
	}

	return name, q
}

const qValuePrefix = "q="

// parseQValue parses an RFC 7231 q-value: 0.0 to 1.0, up to 3 decimal digits.
// We don't import strconv.ParseFloat to keep this allocation-free.
//
// Returns the parsed q-value and nil on success, or 0 and one of the
// static err* values on failure.
func parseQValue(input string) (float64, error) {
	if input == "" {
		return 0, errEmptyQValue
	}

	neg, pos := parseQValueSign(input)
	intPart, newPos, ok := parseQValueInt(input, pos)

	if !ok {
		return 0, fmt.Errorf("%w: %q", errInvalidQInt, input)
	}

	pos = newPos

	frac, fracDiv, newPos := parseQValueFrac(input, pos)
	pos = newPos

	if pos != len(input) {
		return 0, fmt.Errorf("%w: %q", errTrailingQChars, input)
	}

	if intPart == 1 && frac > 0 {
		return 0, fmt.Errorf("%w: %q", errQValueTooLarge, input)
	}

	return composeQValue(intPart, frac, fracDiv, neg), nil
}

const qValueSignChar = '-'

// parseQValueSign consumes an optional sign character from the start of s
// and returns the sign and the new position. The "+" sign is accepted but
// has no effect (RFC 7231 doesn't allow negative q-values, but we don't
// reject them here — we just preserve the sign).
func parseQValueSign(s string) (bool, int) {
	if len(s) > 0 && (s[0] == qValueSignChar || s[0] == '+') {
		return s[0] == qValueSignChar, 1
	}

	return false, 0
}

// parseQValueInt consumes the integer part of a q-value (which per RFC 7231
// is restricted to "0" or "1"). Returns the integer value, the new position,
// and ok=false if the integer part is missing or out of range.
func parseQValueInt(s string, pos int) (int, int, bool) {
	if pos >= len(s) || s[pos] < '0' || s[pos] > '1' {
		return 0, pos, false
	}

	return int(s[pos] - '0'), pos + 1, true
}

const (
	qValueFracBase      = 10
	qValueFracMaxDigits = 3
)

// parseQValueFrac consumes the optional fractional part of a q-value
// (e.g., ".8" in "0.8"). Returns the accumulated numerator, denominator
// (10^N for N digits), and the new position. Always succeeds; absence of
// the fractional part returns (0, 1, pos).
func parseQValueFrac(input string, pos int) (int, int, int) {
	if pos >= len(input) || input[pos] != '.' {
		return 0, 1, pos
	}

	pos++

	frac := 0
	fracDiv := 1
	digits := 0

	for digits < qValueFracMaxDigits && pos < len(input) && input[pos] >= '0' && input[pos] <= '9' {
		frac = frac*qValueFracBase + int(input[pos]-'0')
		fracDiv *= qValueFracBase
		digits++

		pos++
	}

	return frac, fracDiv, pos
}

// composeQValue assembles a q-value from its integer and fractional parts.
func composeQValue(intPart, frac, fracDiv int, neg bool) float64 {
	val := float64(intPart) + float64(frac)/float64(fracDiv)

	if neg {
		val = -val
	}

	return val
}

// trim removes leading and trailing ASCII whitespace without allocating.
func trim(input string) string {
	start := 0
	end := len(input)

	for start < end && isSpace(input[start]) {
		start++
	}

	for end > start && isSpace(input[end-1]) {
		end--
	}

	return input[start:end]
}

// isSpace reports whether b is an HTTP header whitespace character.
func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

// indexByte is a local version of strings.IndexByte to avoid the import.
func indexByte(input string, target byte) int {
	for i := range len(input) {
		if input[i] == target {
			return i
		}
	}

	return -1
}

// indexOf returns the index of target in list, or -1 if not found.
// Linear scan; n is tiny (1-5 encodings).
func indexOf(list []string, target string) int {
	for i, item := range list {
		if item == target {
			return i
		}
	}

	return -1
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
	if len(cfg.WriterFactories) == 0 {
		cfg.WriterFactories = DefaultWriterFactories()
	}

	neg := buildNegotiator(cfg.WriterFactories)

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

			cw := newCompressWriter(resp, cfg.MinSize, encoding, factory)
			defer func() { _ = cw.Close() }()

			next.ServeHTTP(cw, req)
		})
	}
}
