package httputil

import (
	"testing"
)

func TestEntityTag_String_Strong(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc123", EntityTagStrong)

	if got := tag.String(); got != `"abc123"` {
		t.Errorf("String() = %q, want %q", got, `"abc123"`)
	}
}

func TestEntityTag_String_Weak(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc123", EntityTagWeak)

	if got := tag.String(); got != `W/"abc123"` {
		t.Errorf("String() = %q, want %q", got, `W/"abc123"`)
	}
}

func TestEntityTag_String_EmptyOpaqueStrong(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("", EntityTagStrong)

	if got := tag.String(); got != `""` {
		t.Errorf("String() = %q, want %q", got, `""`)
	}
}

func TestEntityTag_OpaqueTag(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("my-value", EntityTagStrong)

	if got := tag.OpaqueTag(); got != "my-value" {
		t.Errorf("OpaqueTag() = %q, want %q", got, "my-value")
	}
}

func TestEntityTag_IsWeak_StrongReportsFalse(t *testing.T) {
	t.Parallel()

	if NewEntityTag("x", EntityTagStrong).IsWeak() {
		t.Error("EntityTagStrong tag reports IsWeak() = true")
	}
}

func TestEntityTag_IsWeak_WeakReportsTrue(t *testing.T) {
	t.Parallel()

	if !NewEntityTag("x", EntityTagWeak).IsWeak() {
		t.Error("EntityTagWeak tag reports IsWeak() = false")
	}
}

func TestEntityTag_IsValid_ZeroValueIsInvalid(t *testing.T) {
	t.Parallel()

	var empty EntityTag

	if empty.IsValid() {
		t.Error("zero-value EntityTag reports IsValid() = true")
	}
}

func TestEntityTag_IsValid_WithOpaqueIsValid(t *testing.T) {
	t.Parallel()

	if !NewEntityTag("x", EntityTagStrong).IsValid() {
		t.Error("EntityTag with opaque 'x' reports IsValid() = false")
	}
}

func TestEntityTag_Comparison_BothWeakSameOpaque(t *testing.T) {
	t.Parallel()

	a := NewEntityTag("1", EntityTagWeak)
	b := NewEntityTag("1", EntityTagWeak)

	if a.StrongEqual(b) {
		t.Error("StrongEqual() = true, want false for both weak")
	}

	if !a.WeakEqual(b) {
		t.Error("WeakEqual() = false, want true for same opaque")
	}
}

func TestEntityTag_Comparison_WeakVsStrongSameOpaque(t *testing.T) {
	t.Parallel()

	a := NewEntityTag("1", EntityTagWeak)
	b := NewEntityTag("1", EntityTagStrong)

	if a.StrongEqual(b) {
		t.Error("StrongEqual() = true, want false when one is weak")
	}

	if !a.WeakEqual(b) {
		t.Error("WeakEqual() = false, want true for same opaque")
	}
}

func TestEntityTag_Comparison_BothStrongSameOpaque(t *testing.T) {
	t.Parallel()

	a := NewEntityTag("1", EntityTagStrong)
	b := NewEntityTag("1", EntityTagStrong)

	if !a.StrongEqual(b) {
		t.Error("StrongEqual() = false, want true for both strong same opaque")
	}

	if !a.WeakEqual(b) {
		t.Error("WeakEqual() = false, want true for same opaque")
	}
}

func TestEntityTag_Comparison_BothStrongDifferentOpaque(t *testing.T) {
	t.Parallel()

	a := NewEntityTag("1", EntityTagStrong)
	b := NewEntityTag("2", EntityTagStrong)

	if a.StrongEqual(b) {
		t.Error("StrongEqual() = true, want false for different opaque")
	}

	if a.WeakEqual(b) {
		t.Error("WeakEqual() = true, want false for different opaque")
	}
}

func TestParseEntityTag_StrongTag(t *testing.T) {
	t.Parallel()

	tag, ok := ParseEntityTag(`"abc"`)
	if !ok {
		t.Fatal(`ParseEntityTag("abc") ok = false, want true`)
	}

	if tag.OpaqueTag() != "abc" {
		t.Errorf("opaque = %q, want %q", tag.OpaqueTag(), "abc")
	}

	if tag.IsWeak() {
		t.Error("weak = true, want false")
	}
}

func TestParseEntityTag_WeakTag(t *testing.T) {
	t.Parallel()

	tag, ok := ParseEntityTag(`W/"abc"`)
	if !ok {
		t.Fatal(`ParseEntityTag(W/"abc") ok = false, want true`)
	}

	if tag.OpaqueTag() != "abc" {
		t.Errorf("opaque = %q, want %q", tag.OpaqueTag(), "abc")
	}

	if !tag.IsWeak() {
		t.Error("weak = false, want true")
	}
}

func TestParseEntityTag_WithWhitespace(t *testing.T) {
	t.Parallel()

	tag, ok := ParseEntityTag(`  "abc"  `)
	if !ok {
		t.Fatal(`ParseEntityTag(  "abc"  ) ok = false, want true`)
	}

	if tag.OpaqueTag() != "abc" {
		t.Errorf("opaque = %q, want %q", tag.OpaqueTag(), "abc")
	}
}

func TestParseEntityTag_NoQuotes(t *testing.T) {
	t.Parallel()

	_, ok := ParseEntityTag(`abc`)
	if ok {
		t.Error(`ParseEntityTag(abc) ok = true, want false`)
	}
}

func TestParseEntityTag_EmptyQuotes(t *testing.T) {
	t.Parallel()

	_, ok := ParseEntityTag(`""`)
	if ok {
		t.Error(`ParseEntityTag("") ok = true, want false`)
	}
}

func TestParseEntityTag_OnlyOpeningQuote(t *testing.T) {
	t.Parallel()

	_, ok := ParseEntityTag(`"abc`)
	if ok {
		t.Error(`ParseEntityTag("abc) ok = true, want false`)
	}
}

func TestParseEntityTag_OnlyClosingQuote(t *testing.T) {
	t.Parallel()

	_, ok := ParseEntityTag(`abc"`)
	if ok {
		t.Error(`ParseEntityTag(abc") ok = true, want false`)
	}
}

func TestParseEntityTag_EmptyString(t *testing.T) {
	t.Parallel()

	_, ok := ParseEntityTag(``)
	if ok {
		t.Error(`ParseEntityTag() ok = true, want false`)
	}
}

func TestParseEntityTag_WSlashOnly(t *testing.T) {
	t.Parallel()

	_, ok := ParseEntityTag(`W/`)
	if ok {
		t.Error(`ParseEntityTag(W/) ok = true, want false`)
	}
}

func TestParseEntityTagList(t *testing.T) {
	t.Parallel()

	tags := ParseEntityTagList(`"a,b", W/"c", "d"`)
	if len(tags) != 3 {
		t.Fatalf("len(tags) = %d, want 3", len(tags))
	}

	if tags[0].OpaqueTag() != "a,b" {
		t.Errorf("tags[0] opaque = %q, want %q", tags[0].OpaqueTag(), "a,b")
	}

	if !tags[1].IsWeak() {
		t.Error("tags[1] should be weak")
	}

	if tags[1].OpaqueTag() != "c" {
		t.Errorf("tags[1] opaque = %q, want %q", tags[1].OpaqueTag(), "c")
	}

	if tags[2].OpaqueTag() != "d" {
		t.Errorf("tags[2] opaque = %q, want %q", tags[2].OpaqueTag(), "d")
	}
}

func TestParseEntityTagList_SkipsMalformed(t *testing.T) {
	t.Parallel()

	tags := ParseEntityTagList(`"valid", not-a-tag, "also-valid"`)
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2 (malformed tag skipped)", len(tags))
	}
}

func TestParseEntityTagList_Empty(t *testing.T) {
	t.Parallel()

	tags := ParseEntityTagList("")
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

func TestSplitRawEntityTags_EscapedQuotes(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`"a\"b", "c"`)

	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	want0 := `"a\"b"`
	if tags[0] != want0 {
		t.Errorf("tags[0] = %q, want %q", tags[0], want0)
	}

	want1 := `"c"`
	if tags[1] != want1 {
		t.Errorf("tags[1] = %q, want %q", tags[1], want1)
	}
}

func TestSplitRawEntityTags_TrailingComma(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`"a", "b",`)
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	if tags[0] != `"a"` {
		t.Errorf("tags[0] = %q, want %q", tags[0], `"a"`)
	}
}

func TestSplitRawEntityTags_LeadingComma(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`, "a", "b"`)
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	if tags[0] != `"a"` {
		t.Errorf("tags[0] = %q, want %q", tags[0], `"a"`)
	}
}

func TestSplitRawEntityTags_EmptyEntries(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`"a",, "b"`)
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	if tags[0] != `"a"` {
		t.Errorf("tags[0] = %q, want %q", tags[0], `"a"`)
	}
}

func TestSplitRawEntityTags_WhitespaceBetweenCommas(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`"a",   , "b"`)
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	if tags[0] != `"a"` {
		t.Errorf("tags[0] = %q, want %q", tags[0], `"a"`)
	}
}

func TestSplitRawEntityTags_SingleTagNoComma(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`"only"`)
	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1", len(tags))
	}

	if tags[0] != `"only"` {
		t.Errorf("tags[0] = %q, want %q", tags[0], `"only"`)
	}
}

func TestSplitRawEntityTags_OnlyCommas(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`,,,`)
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

func TestSplitRawEntityTags_OnlyWhitespace(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`   `)
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

func TestSplitRawEntityTags_CommaInOpaqueTag(t *testing.T) {
	t.Parallel()

	tags := splitRawEntityTags(`"a,b,c"`)
	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1", len(tags))
	}

	if tags[0] != `"a,b,c"` {
		t.Errorf("tags[0] = %q, want %q", tags[0], `"a,b,c"`)
	}
}

func TestParseEntityTagList_AllMalformed(t *testing.T) {
	t.Parallel()

	tags := ParseEntityTagList("garbage, also-not-a-tag, 12345")
	if tags == nil {
		t.Fatal("ParseEntityTagList returned nil for all-malformed input")
	}

	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0 for all-malformed input", len(tags))
	}
}

func TestParseEntityTagList_WhitespaceOnly(t *testing.T) {
	t.Parallel()

	tags := ParseEntityTagList("   ")
	if tags == nil {
		t.Fatal("ParseEntityTagList returned nil for whitespace-only input")
	}

	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

func TestMatchesIfNoneMatch_Wildcard(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfNoneMatch(tag, "*") {
		t.Error("MatchesIfNoneMatch() = false, want true for wildcard")
	}
}

func TestMatchesIfNoneMatch_ExactMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfNoneMatch(tag, `"abc"`) {
		t.Error("MatchesIfNoneMatch() = false, want true for exact match")
	}
}

func TestMatchesIfNoneMatch_WeakMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfNoneMatch(tag, `W/"abc"`) {
		t.Error("MatchesIfNoneMatch() = false, want true for weak match")
	}
}

func TestMatchesIfNoneMatch_ListContainsMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfNoneMatch(tag, `"other", "abc", "third"`) {
		t.Error("MatchesIfNoneMatch() = false, want true when list contains match")
	}
}

func TestMatchesIfNoneMatch_ListContainsWeakMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfNoneMatch(tag, `"other", W/"abc"`) {
		t.Error("MatchesIfNoneMatch() = false, want true when list contains weak match")
	}
}

func TestMatchesIfNoneMatch_NoMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if MatchesIfNoneMatch(tag, `"different"`) {
		t.Error("MatchesIfNoneMatch() = true, want false for no match")
	}
}

func TestMatchesIfNoneMatch_EmptyHeader(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if MatchesIfNoneMatch(tag, "") {
		t.Error("MatchesIfNoneMatch() = true, want false for empty header")
	}
}

func TestMatchesIfMatch_Wildcard(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfMatch(tag, "*") {
		t.Error("MatchesIfMatch() = false, want true for wildcard")
	}
}

func TestMatchesIfMatch_StrongMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfMatch(tag, `"abc"`) {
		t.Error("MatchesIfMatch() = false, want true for strong match")
	}
}

func TestMatchesIfMatch_ListContainsStrongMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if !MatchesIfMatch(tag, `"other", "abc"`) {
		t.Error("MatchesIfMatch() = false, want true when list contains strong match")
	}
}

func TestMatchesIfMatch_WeakTagDoesNotStronglyMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if MatchesIfMatch(tag, `W/"abc"`) {
		t.Error("MatchesIfMatch() = true, want false for weak tag (strong comparison)")
	}
}

func TestMatchesIfMatch_NoMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if MatchesIfMatch(tag, `"different"`) {
		t.Error("MatchesIfMatch() = true, want false for no match")
	}
}

func TestMatchesIfMatch_EmptyHeader(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	if MatchesIfMatch(tag, "") {
		t.Error("MatchesIfMatch() = true, want false for empty header")
	}
}

func TestMatchesIfMatch_WeakServerTag(t *testing.T) {
	t.Parallel()

	// A weak server tag can never strongly match any If-Match value.
	weakTag := NewEntityTag("abc", EntityTagWeak)

	if MatchesIfMatch(weakTag, `"abc"`) {
		t.Error("weak server tag should not strongly match")
	}
}

func FuzzParseEntityTag(f *testing.F) {
	f.Add(`"abc"`)
	f.Add(`W/"abc"`)
	f.Add("abc")
	f.Add("")
	f.Add(`""`)
	f.Add(`"a,b"`)
	f.Add(`"a\"b"`)
	f.Add("*")
	f.Add(`  "abc"  `)
	f.Add(`"W/abc"`)
	f.Add(`W/W/"abc"`)

	f.Fuzz(func(t *testing.T, raw string) {
		tag, ok := ParseEntityTag(raw)

		if !ok {
			if tag.IsValid() {
				t.Fatalf("ParseEntityTag(%q) returned ok=false but tag is valid: %v", raw, tag)
			}

			return
		}

		if !tag.IsValid() {
			t.Fatalf("ParseEntityTag(%q) returned ok=true but tag is not valid", raw)
		}

		reparsed, reparseOK := ParseEntityTag(tag.String())
		if !reparseOK {
			t.Fatalf("ParseEntityTag(%q).String() = %q failed to re-parse", raw, tag.String())
		}

		if reparsed != tag {
			t.Fatalf("round-trip mismatch: %v != %v", reparsed, tag)
		}
	})
}

func FuzzParseEntityTagList(f *testing.F) {
	f.Add(`"abc"`)
	f.Add(`"abc", "def"`)
	f.Add(`"abc", W/"def"`)
	f.Add("*")
	f.Add("")
	f.Add(`"abc",, "def"`)
	f.Add(`"a,b"`)
	f.Add(`"a\"b", "c"`)
	f.Add(`W/"x", "y", W/"z"`)
	f.Add(`"unterminated`)

	f.Fuzz(func(t *testing.T, header string) {
		tags := ParseEntityTagList(header)

		if tags == nil {
			t.Fatalf("ParseEntityTagList(%q) returned nil; expected non-nil slice", header)
		}

		for i, tag := range tags {
			if !tag.IsValid() {
				t.Fatalf("ParseEntityTagList(%q)[%d] is not a valid EntityTag: %v", header, i, tag)
			}

			reparsed, ok := ParseEntityTag(tag.String())
			if !ok || reparsed != tag {
				t.Fatalf("ParseEntityTagList(%q)[%d] round-trip failed: %v", header, i, tag)
			}
		}
	})
}
