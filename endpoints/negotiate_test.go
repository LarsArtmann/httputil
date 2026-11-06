package endpoints

import "testing"

func TestNegotiateRepresentation_EmptyHeader_SelectsJSON(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation(""); got != representationJSON {
		t.Errorf("negotiateRepresentation(\"\") = %v, want representationJSON", got)
	}
}

func TestNegotiateRepresentation_ExplicitHTML_SelectsHTML(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("text/html"); got != representationHTML {
		t.Errorf("got = %v, want representationHTML", got)
	}
}

func TestNegotiateRepresentation_ExplicitJSON_SelectsJSON(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("application/json"); got != representationJSON {
		t.Errorf("got = %v, want representationJSON", got)
	}
}

func TestNegotiateRepresentation_WildcardAny_SelectsJSON(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("*/*"); got != representationJSON {
		t.Errorf("got = %v, want representationJSON", got)
	}
}

func TestNegotiateRepresentation_TextWildcard_SelectsHTML(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("text/*"); got != representationHTML {
		t.Errorf("got = %v, want representationHTML", got)
	}
}

func TestNegotiateRepresentation_ApplicationWildcard_SelectsJSON(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("application/*"); got != representationJSON {
		t.Errorf("got = %v, want representationJSON", got)
	}
}

func TestNegotiateRepresentation_BrowserHeader_SelectsHTML(t *testing.T) {
	t.Parallel()

	browser := "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"

	if got := negotiateRepresentation(browser); got != representationHTML {
		t.Errorf("got = %v, want representationHTML for a browser Accept header", got)
	}
}

func TestNegotiateRepresentation_HigherQForJSON_SelectsJSON(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("text/html;q=0.5, application/json;q=0.9"); got != representationJSON {
		t.Errorf("got = %v, want representationJSON", got)
	}
}

func TestNegotiateRepresentation_HigherQForHTML_SelectsHTML(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("text/html;q=0.9, application/json;q=0.5"); got != representationHTML {
		t.Errorf("got = %v, want representationHTML", got)
	}
}

func TestNegotiateRepresentation_TieBreaksTowardJSON(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("text/html, application/json"); got != representationJSON {
		t.Errorf("got = %v, want representationJSON (ties break toward the machine default)", got)
	}
}

func TestNegotiateRepresentation_ZeroQExcludesHTML(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("text/html;q=0, application/json;q=0.1"); got != representationJSON {
		t.Errorf("got = %v, want representationJSON (q=0 excludes HTML)", got)
	}
}

func TestNegotiateRepresentation_UnacceptableEverything_SelectsJSON(t *testing.T) {
	t.Parallel()

	// Documented never-406 posture: a header that excludes both supported
	// representations falls back to the machine default rather than failing.
	if got := negotiateRepresentation("*/*;q=0"); got != representationJSON {
		t.Errorf("got = %v, want representationJSON (never 406)", got)
	}
}

func TestNegotiateRepresentation_GarbageEntriesAreSkipped(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("bogus;;, text/html, ;;;"); got != representationHTML {
		t.Errorf("got = %v, want representationHTML (garbage skipped, html seen)", got)
	}
}

func TestNegotiateRepresentation_MatchIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	if got := negotiateRepresentation("TEXT/HTML"); got != representationHTML {
		t.Errorf("got = %v, want representationHTML", got)
	}
}

func TestNegotiateRepresentation_UnparseableQFallsBackToOne(t *testing.T) {
	t.Parallel()

	// The lenient posture: a broken q-parameter falls back to the default
	// weight, so text/html;q=banana beats application/json;q=0.9.
	if got := negotiateRepresentation("text/html;q=banana, application/json;q=0.9"); got != representationHTML {
		t.Errorf("got = %v, want representationHTML", got)
	}
}

func TestNegotiateRepresentation_IgnoresAcceptExtensions(t *testing.T) {
	t.Parallel()

	// RFC 9110 example shape: parameters other than q are accept-ext.
	if got := negotiateRepresentation(
		"text/*;q=0.3, text/html;q=0.7, text/html;level=1, */*;q=0.5",
	); got != representationHTML {
		t.Errorf("got = %v, want representationHTML", got)
	}
}

func TestParseMediaRange_ReturnsLowercasedType(t *testing.T) {
	t.Parallel()

	mediaRange, weight, ok := parseMediaRange("  TEXT/html ; Q=0.9 ")

	if !ok {
		t.Fatal("ok = false, want true")
	}

	if mediaRange != "text/html" {
		t.Errorf("mediaRange = %q, want %q", mediaRange, "text/html")
	}

	if weight != 0.9 {
		t.Errorf("weight = %v, want 0.9", weight)
	}
}

func TestParseMediaRange_NoSlash_ReturnsNotOK(t *testing.T) {
	t.Parallel()

	_, _, ok := parseMediaRange("garbage")

	if ok {
		t.Error("ok = true, want false for an entry without '/'")
	}
}

func TestParseMediaRange_MissingQDefaultsToOne(t *testing.T) {
	t.Parallel()

	mediaRange, weight, ok := parseMediaRange("application/json;charset=utf8")

	if !ok || mediaRange != "application/json" {
		t.Fatalf("got (%q, %v), want (application/json, true)", mediaRange, ok)
	}

	if weight != defaultQValue {
		t.Errorf("weight = %v, want %v", weight, defaultQValue)
	}
}
