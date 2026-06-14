package httputil

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
