package httputil

import (
	"math/rand"
	"strings"
	"testing"
)

// negotiationQChoices are the q-values the property-test generator emits.
// They are exact float64 literals so the reference model and the parser
// (frac/division) produce bit-identical values for comparison.
var negotiationQChoices = []struct {
	raw string
	q   float64
}{
	{raw: "", q: defaultQValue},
	{raw: "q=0", q: 0},
	{raw: "q=0.001", q: 0.001},
	{raw: "q=0.25", q: 0.25},
	{raw: "q=0.5", q: 0.5},
	{raw: "q=0.874", q: 0.874},
	{raw: "q=1", q: 1},
}

// negotiationNamePool lists encodings the generator may place in a header:
// everything the test negotiator registers (gzip, deflate, identity) plus
// tokens it must never select (unregistered names and the "*" wildcard).
var negotiationNamePool = []string{
	encodingGzip,
	encodingDeflate,
	encodingIdentity,
	encodingBr,
	encodingZstd,
	"*",
	"bzip2",
}

// generatedEntry is one structured Accept-Encoding entry: the model uses
// name/q directly while the header carries the rendered wire form.
type generatedEntry struct {
	name string
	q    float64
	raw  string
}

// generateNegotiationHeader builds a random header plus its structured form.
func generateNegotiationHeader(rng *rand.Rand) (string, []generatedEntry) {
	count := 1 + rng.Intn(6)
	parts := make([]string, 0, count)
	entries := make([]generatedEntry, 0, count)

	for range count {
		name := negotiationNamePool[rng.Intn(len(negotiationNamePool))]
		if rng.Intn(4) == 0 {
			name = strings.ToUpper(name[:1]) + name[1:]
		}

		choice := negotiationQChoices[rng.Intn(len(negotiationQChoices))]

		raw := name
		if choice.raw != "" {
			raw = name + ";" + choice.raw
		}

		if rng.Intn(2) == 0 {
			raw = " " + raw
		}

		if rng.Intn(2) == 0 {
			raw += " "
		}

		parts = append(parts, raw)
		entries = append(entries, generatedEntry{name: strings.ToLower(name), q: choice.q, raw: raw})
	}

	return strings.Join(parts, ","), entries
}

// modelNegotiation is the reference selection: among entries the negotiator
// has registered with q > 0, pick the maximum q; break ties by server
// preference order. With no candidate, fall back to identity when registered.
func modelNegotiation(order []string, factories map[string]bool, entries []generatedEntry) (string, float64, bool) {
	bestIdx := -1
	bestQ := -1.0
	bestOrder := len(order) + 1

	for i, entry := range entries {
		if !factories[entry.name] || entry.q <= 0 {
			continue
		}

		orderIdx := indexOf(order, entry.name)
		if orderIdx < 0 {
			continue
		}

		if entry.q > bestQ || (entry.q == bestQ && orderIdx < bestOrder) {
			bestIdx = i
			bestQ = entry.q
			bestOrder = orderIdx
		}
	}

	if bestIdx >= 0 {
		return entries[bestIdx].name, entries[bestIdx].q, true
	}

	if factories[encodingIdentity] {
		return encodingIdentity, defaultQValue, true
	}

	return "", 0, false
}

func testNegotiatorFacts(neg *negotiator) ([]string, map[string]bool) {
	available := make(map[string]bool, len(neg.order))

	for _, name := range neg.order {
		available[name] = true
	}

	return neg.order, available
}

func TestNegotiator_Property_SelectsHighestQAvailable(t *testing.T) {
	t.Parallel()

	src := rand.NewSource(20260830)
	rng := rand.New(src) //nolint:gosec // G404: deterministic property-test generator, not security
	neg := newTestNegotiator()
	order, available := testNegotiatorFacts(neg)

	const qEpsilon = 1e-9

	for range 2000 {
		header, entries := generateNegotiationHeader(rng)

		gotName, gotQ, gotOK := neg.negotiateEncoding(header)
		wantName, wantQ, wantOK := modelNegotiation(order, available, entries)

		if gotOK != wantOK || gotName != wantName {
			t.Fatalf("header %q: got (%q, %v), want (%q, %v)", header, gotName, gotOK, wantName, wantOK)
		}

		if gotQ < wantQ-qEpsilon || gotQ > wantQ+qEpsilon {
			t.Fatalf("header %q: got q %f, want %f", header, gotQ, wantQ)
		}

		if gotOK && !available[gotName] {
			t.Fatalf("header %q: selected unavailable encoding %q", header, gotName)
		}
	}
}

func TestNegotiator_Property_HeaderOrderInvariance(t *testing.T) {
	t.Parallel()

	src := rand.NewSource(20260831)
	rng := rand.New(src) //nolint:gosec // G404: deterministic property-test generator, not security
	neg := newTestNegotiator()

	for range 2000 {
		header, entries := generateNegotiationHeader(rng)

		wantName, wantQ, wantOK := neg.negotiateEncoding(header)

		for range 3 {
			rng.Shuffle(len(entries), func(i, j int) {
				entries[i], entries[j] = entries[j], entries[i]
			})

			shuffledParts := make([]string, 0, len(entries))
			for _, entry := range entries {
				shuffledParts = append(shuffledParts, entry.raw)
			}

			shuffled := strings.Join(shuffledParts, ",")

			gotName, gotQ, gotOK := neg.negotiateEncoding(shuffled)

			if gotOK != wantOK || gotName != wantName || gotQ != wantQ {
				t.Fatalf(
					"reordering changed the outcome: %q -> (%q %f %v), %q -> (%q %f %v)",
					header, wantName, wantQ, wantOK, shuffled, gotName, gotQ, gotOK,
				)
			}
		}
	}
}
