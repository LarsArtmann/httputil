package httputil

import (
	"strings"
)

// parseEncodingEntry parses a single entry like "gzip" or "gzip;q=0.8"
// and returns the canonical encoding name and q-value.
func parseEncodingEntry(entry string) (string, float64) {
	name, rest, found := strings.Cut(entry, ";")
	name = strings.ToLower(trim(name))
	rest = trim(rest)

	if !found || !strings.HasPrefix(rest, qValuePrefix) {
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
		return 0, errInvalidQInt.WithContext("input", input)
	}

	pos = newPos

	frac, fracDiv, newPos := parseQValueFrac(input, pos)
	pos = newPos

	if pos != len(input) {
		return 0, errTrailingQChars.WithContext("input", input)
	}

	if intPart == 1 && frac > 0 {
		return 0, errQValueTooLarge.WithContext("input", input)
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
