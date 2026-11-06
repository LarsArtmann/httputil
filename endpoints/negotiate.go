package endpoints

import "strings"

// representation identifies one of the listing's two wire forms, selected by
// proactive content negotiation (RFC 9110) on the Accept request header.
type representation int

const (
	representationJSON representation = iota
	representationHTML
)

// qUnacceptable marks a representation no Accept range made acceptable; a
// q-value of 0 excludes rather than selects, so it never raises a best above
// this floor.
const qUnacceptable = -1.0

// defaultQValue is the q-value of an Accept entry without an explicit q
// parameter (RFC 9110: "The default value is 1").
const defaultQValue = 1.0

// negotiateRepresentation selects the listing representation for an Accept
// header value. Supported ranges, most specific first:
//
//   - "text/html" selects HTML; "application/json" selects JSON.
//   - "text/*" selects HTML; "application/*" and "*/*" select JSON.
//
// The representation whose best range carries the highest q-value wins; ties
// break toward JSON, the machine default. An absent or empty header selects
// JSON, as does a header nothing acceptable was found in — this listing never
// answers 406, mirroring the root module's Language middleware, which falls
// back to its default tag instead of failing the request. Unparseable entries
// are skipped (lenient posture, the same stance as the root module's
// compression and Accept-Language parsers).
func negotiateRepresentation(accept string) representation {
	if accept == "" {
		return representationJSON
	}

	bestJSON := qUnacceptable
	bestHTML := qUnacceptable

	for entry := range strings.SplitSeq(accept, ",") {
		mediaRange, weight, ok := parseMediaRange(entry)
		if !ok || weight <= 0 {
			continue
		}

		switch mediaRange {
		case "text/html", "text/*":
			bestHTML = max(bestHTML, weight)
		case "application/json", "application/*", "*/*":
			bestJSON = max(bestJSON, weight)
		}
	}

	if bestHTML > bestJSON {
		return representationHTML
	}

	return representationJSON
}

// parseMediaRange parses one Accept entry ("text/html;q=0.9" yields
// "text/html", 0.9, true). Parameters other than q are accept extensions and
// ignored; the q-value defaults to defaultQValue when absent, and an
// unparseable q-value falls back to it (lenient posture). ok is false for
// entries without a '/', which cannot be a media range.
func parseMediaRange(entry string) (string, float64, bool) {
	mediaType, params, _ := strings.Cut(entry, ";")
	mediaType = strings.ToLower(trimOWS(mediaType))

	if !strings.Contains(mediaType, "/") {
		return "", 0, false
	}

	for param := range strings.SplitSeq(params, ";") {
		name, value, _ := strings.Cut(param, "=")
		if strings.ToLower(trimOWS(name)) != qParamName {
			continue
		}

		weight, err := parseQValue(trimOWS(value))
		if err != nil {
			return mediaType, defaultQValue, true
		}

		return mediaType, weight, true
	}

	return mediaType, defaultQValue, true
}

// qParamName is the Accept parameter that carries the quality value.
const qParamName = "q"

// trimOWS removes leading and trailing optional whitespace (RFC 9110 OWS:
// SP or HTAB) without allocating.
func trimOWS(input string) string {
	start := 0
	end := len(input)

	for start < end && isOWS(input[start]) {
		start++
	}

	for end > start && isOWS(input[end-1]) {
		end--
	}

	return input[start:end]
}

// isOWS reports whether b is an RFC 9110 optional-whitespace byte.
func isOWS(b byte) bool {
	return b == ' ' || b == '\t'
}
