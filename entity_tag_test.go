package httputil

import (
	"testing"
)

func TestEntityTag_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tag  EntityTag
		want string
	}{
		{
			name: "strong",
			tag:  NewEntityTag("abc123", EntityTagStrong),
			want: `"abc123"`,
		},
		{
			name: "weak",
			tag:  NewEntityTag("abc123", EntityTagWeak),
			want: `W/"abc123"`,
		},
		{
			name: "empty opaque strong",
			tag:  NewEntityTag("", EntityTagStrong),
			want: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.tag.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEntityTag_OpaqueTag(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("my-value", EntityTagStrong)

	if got := tag.OpaqueTag(); got != "my-value" {
		t.Errorf("OpaqueTag() = %q, want %q", got, "my-value")
	}
}

func TestEntityTag_IsWeak(t *testing.T) {
	t.Parallel()

	if NewEntityTag("x", EntityTagStrong).IsWeak() {
		t.Error("EntityTagStrong tag reports IsWeak() = true")
	}

	if !NewEntityTag("x", EntityTagWeak).IsWeak() {
		t.Error("EntityTagWeak tag reports IsWeak() = false")
	}
}

func TestEntityTag_IsValid(t *testing.T) {
	t.Parallel()

	var empty EntityTag
	if empty.IsValid() {
		t.Error("zero-value EntityTag reports IsValid() = true")
	}

	if !NewEntityTag("x", EntityTagStrong).IsValid() {
		t.Error("EntityTag with opaque 'x' reports IsValid() = false")
	}
}

func TestEntityTag_Comparison(t *testing.T) {
	t.Parallel()

	// RFC 7232 §2.3.2 comparison table: all four combinations of
	// strong/weak x matching/non-matching opaque-tags.
	tests := []struct {
		name   string
		a      EntityTag
		b      EntityTag
		strong bool
		weak   bool
	}{
		{
			name:   "both weak same opaque",
			a:      NewEntityTag("1", EntityTagWeak),
			b:      NewEntityTag("1", EntityTagWeak),
			strong: false,
			weak:   true,
		},
		{
			name:   "weak vs strong same opaque",
			a:      NewEntityTag("1", EntityTagWeak),
			b:      NewEntityTag("1", EntityTagStrong),
			strong: false,
			weak:   true,
		},
		{
			name:   "both strong same opaque",
			a:      NewEntityTag("1", EntityTagStrong),
			b:      NewEntityTag("1", EntityTagStrong),
			strong: true,
			weak:   true,
		},
		{
			name:   "both strong different opaque",
			a:      NewEntityTag("1", EntityTagStrong),
			b:      NewEntityTag("2", EntityTagStrong),
			strong: false,
			weak:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.StrongEqual(tt.b); got != tt.strong {
				t.Errorf("StrongEqual() = %v, want %v", got, tt.strong)
			}

			if got := tt.a.WeakEqual(tt.b); got != tt.weak {
				t.Errorf("WeakEqual() = %v, want %v", got, tt.weak)
			}
		})
	}
}

func TestParseEntityTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantOk     bool
		wantOpaque string
		wantWeak   bool
	}{
		{name: "strong tag", input: `"abc"`, wantOk: true, wantOpaque: "abc", wantWeak: false},
		{name: "weak tag", input: `W/"abc"`, wantOk: true, wantOpaque: "abc", wantWeak: true},
		{name: "with whitespace", input: `  "abc"  `, wantOk: true, wantOpaque: "abc", wantWeak: false},
		{name: "no quotes", input: `abc`, wantOk: false},
		{name: "empty quotes", input: `""`, wantOk: false},
		{name: "only opening quote", input: `"abc`, wantOk: false},
		{name: "only closing quote", input: `abc"`, wantOk: false},
		{name: "empty string", input: ``, wantOk: false},
		{name: "W/ only", input: `W/`, wantOk: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tag, ok := ParseEntityTag(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("ParseEntityTag(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}

			if !ok {
				return
			}

			if tag.OpaqueTag() != tt.wantOpaque {
				t.Errorf("opaque = %q, want %q", tag.OpaqueTag(), tt.wantOpaque)
			}

			if tag.IsWeak() != tt.wantWeak {
				t.Errorf("weak = %v, want %v", tag.IsWeak(), tt.wantWeak)
			}
		})
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

func TestSplitRawEntityTags_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		count int
		want0 string
	}{
		{name: "trailing comma", input: `"a", "b",`, count: 2, want0: `"a"`},
		{name: "leading comma", input: `, "a", "b"`, count: 2, want0: `"a"`},
		{name: "empty entries", input: `"a",, "b"`, count: 2, want0: `"a"`},
		{name: "whitespace between commas", input: `"a",   , "b"`, count: 2, want0: `"a"`},
		{name: "single tag no comma", input: `"only"`, count: 1, want0: `"only"`},
		{name: "only commas", input: `,,,`, count: 0},
		{name: "only whitespace", input: `   `, count: 0},
		{name: "comma in opaque tag", input: `"a,b,c"`, count: 1, want0: `"a,b,c"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tags := splitRawEntityTags(tt.input)
			if len(tags) != tt.count {
				t.Fatalf("len(tags) = %d, want %d (input=%q)", len(tags), tt.count, tt.input)
			}

			if tt.count > 0 && tags[0] != tt.want0 {
				t.Errorf("tags[0] = %q, want %q (input=%q)", tags[0], tt.want0, tt.input)
			}
		})
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

func TestMatchesIfNoneMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	tests := []struct {
		name      string
		headerVal string
		want      bool
	}{
		{name: "wildcard", headerVal: "*", want: true},
		{name: "exact match", headerVal: `"abc"`, want: true},
		{name: "weak match", headerVal: `W/"abc"`, want: true},
		{name: "list contains match", headerVal: `"other", "abc", "third"`, want: true},
		{name: "list contains weak match", headerVal: `"other", W/"abc"`, want: true},
		{name: "no match", headerVal: `"different"`, want: false},
		{name: "empty header", headerVal: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := MatchesIfNoneMatch(tag, tt.headerVal); got != tt.want {
				t.Errorf("MatchesIfNoneMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesIfMatch(t *testing.T) {
	t.Parallel()

	tag := NewEntityTag("abc", EntityTagStrong)

	tests := []struct {
		name      string
		headerVal string
		want      bool
	}{
		{name: "wildcard", headerVal: "*", want: true},
		{name: "strong match", headerVal: `"abc"`, want: true},
		{name: "list contains strong match", headerVal: `"other", "abc"`, want: true},
		{name: "weak tag does not strongly match", headerVal: `W/"abc"`, want: false},
		{name: "no match", headerVal: `"different"`, want: false},
		{name: "empty header", headerVal: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := MatchesIfMatch(tag, tt.headerVal); got != tt.want {
				t.Errorf("MatchesIfMatch() = %v, want %v", got, tt.want)
			}
		})
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
