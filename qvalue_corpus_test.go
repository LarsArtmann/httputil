package httputil

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
)

// qValueCorpusPath is the committed corpus that pins the q-value parser's
// semantics. The endpoints sub-module asserts a byte-identical mirror of it,
// so the two deliberately duplicated parsers cannot drift apart silently.
const qValueCorpusPath = "testdata/qvalue-corpus.txt"

const (
	corpusValuePrefix = "value:"
	corpusErrorPrefix = "err:"
)

// qValueCorpusSentinels maps the corpus's error names onto this package's
// sentinels; the name set is shared with the endpoints mirror.
var qValueCorpusSentinels = map[string]error{
	"errEmptyQValue":    errEmptyQValue,
	"errInvalidQInt":    errInvalidQInt,
	"errTrailingQChars": errTrailingQChars,
	"errQValueTooLarge": errQValueTooLarge,
}

// qValueCorpusCase is one corpus row: an input and exactly one expectation.
type qValueCorpusCase struct {
	input   string
	value   float64
	wantErr error
	line    int
}

// TestParseQValue_Corpus locks every documented posture of the q-value
// parser (RFC 9110 grammar, the accepted signs, the lenient empty fraction
// and trailing-zero handling, and each rejection sentinel) to the shared
// corpus that the endpoints copy of this parser is asserted against.
func TestParseQValue_Corpus(t *testing.T) {
	t.Parallel()

	for _, c := range loadQValueCorpus(t) {
		got, err := parseQValue(c.input)
		assertQValueCorpusCase(t, c, got, err)
	}
}

func loadQValueCorpus(t *testing.T) []qValueCorpusCase {
	t.Helper()

	data, err := os.ReadFile(qValueCorpusPath)
	if err != nil {
		t.Fatalf("read q-value corpus %s: %v", qValueCorpusPath, err)
	}

	cases := make([]qValueCorpusCase, 0, strings.Count(string(data), "\n"))

	for index, line := range strings.Split(string(data), "\n") {
		lineNo := index + 1

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			t.Fatalf("%s:%d: want 2 tab-separated fields, got %d: %q", qValueCorpusPath, lineNo, len(fields), line)
		}

		cases = append(cases, newQValueCorpusCase(t, lineNo, fields[0], fields[1]))
	}

	if len(cases) == 0 {
		t.Fatalf("%s: corpus contains no cases", qValueCorpusPath)
	}

	return cases
}

func newQValueCorpusCase(t *testing.T, line int, quotedInput, verdict string) qValueCorpusCase {
	t.Helper()

	input, err := strconv.Unquote(quotedInput)
	if err != nil {
		t.Fatalf("%s:%d: input %q is not a Go-quoted literal: %v", qValueCorpusPath, line, quotedInput, err)
	}

	c := qValueCorpusCase{line: line, input: input}

	switch {
	case strings.HasPrefix(verdict, corpusValuePrefix):
		c.value, err = strconv.ParseFloat(strings.TrimPrefix(verdict, corpusValuePrefix), 64)
		if err != nil {
			t.Fatalf("%s:%d: value verdict %q is not a float: %v", qValueCorpusPath, line, verdict, err)
		}
	case strings.HasPrefix(verdict, corpusErrorPrefix):
		name := strings.TrimPrefix(verdict, corpusErrorPrefix)

		c.wantErr = qValueCorpusSentinels[name]
		if c.wantErr == nil {
			t.Fatalf("%s:%d: unknown sentinel name %q", qValueCorpusPath, line, name)
		}
	default:
		t.Fatalf(
			"%s:%d: verdict %q must start with %q or %q",
			qValueCorpusPath,
			line,
			verdict,
			corpusValuePrefix,
			corpusErrorPrefix,
		)
	}

	return c
}

func assertQValueCorpusCase(t *testing.T, c qValueCorpusCase, got float64, err error) {
	t.Helper()

	if c.wantErr != nil {
		if !errors.Is(err, c.wantErr) {
			t.Errorf("line %d: parseQValue(%q) error = %v, want %v", c.line, c.input, err, c.wantErr)
		}

		return
	}

	if err != nil {
		t.Errorf("line %d: parseQValue(%q) error = %v, want %v", c.line, c.input, err, c.value)

		return
	}

	if got != c.value {
		t.Errorf("line %d: parseQValue(%q) = %v, want %v", c.line, c.input, got, c.value)
	}
}
