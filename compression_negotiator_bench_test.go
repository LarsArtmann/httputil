package httputil

import (
	"testing"
)

// BenchmarkCompressionNegotiator measures the cost of Accept-Encoding
// negotiation per request. The negotiator runs on every request handled by
// Compression, so its throughput directly impacts per-request overhead.
//
// Four header shapes are benchmarked to surface fast-path vs full-parser
// behavior:
//   - singleToken: the common case for non-browser clients ("gzip").
//   - browserMulti: a realistic browser-style multi-encoding header with
//     q-values and whitespace.
//   - qvalues: a pure q-value-heavy header exercising the parser without
//     wildcards or extra whitespace.
//   - emptyHeader: the no-header case (negotiator picks the highest-priority
//     configured encoding).
func BenchmarkCompressionNegotiator(b *testing.B) {
	neg := newTestNegotiator()

	cases := []struct {
		name   string
		header string
	}{
		{name: "singleToken", header: "gzip"},
		{name: "browserMulti", header: "gzip, deflate, br;q=0.9, *;q=0.1"},
		{name: "qvalues", header: "gzip;q=0.1, deflate;q=0.9"},
		{name: "emptyHeader", header: ""},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _, _ = neg.negotiateEncoding(tc.header)
			}
		})
	}
}
