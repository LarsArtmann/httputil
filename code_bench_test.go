package httputil

import "testing"

// Benchmarks for the typed error model: construction via Code constructors,
// sentinel cloning with context, and domain extraction.

func BenchmarkCodeRejectionConstruction(b *testing.B) {
	b.ResetTimer()

	for b.Loop() {
		_ = codeCorsMaxAgeNegative.Rejection("CORSConfig: MaxAge must not be negative")
	}
}

func BenchmarkSentinelCloneWithContext(b *testing.B) {
	b.ResetTimer()

	for b.Loop() {
		_ = errNegativeMaxAge.WithContextAny("max_age", -5)
	}
}

func BenchmarkWrapTransientWithCause(b *testing.B) {
	b.ResetTimer()

	for b.Loop() {
		_ = codeWriteFailed.WrapTransient(errTestBoom, "response writer write failed").
			WithContext("status", "200")
	}
}

func BenchmarkDomainOf(b *testing.B) {
	err := errNegativeMaxAge.WithContextAny("max_age", -5)

	b.ResetTimer()

	for b.Loop() {
		_, _ = DomainOf(err)
	}
}

func BenchmarkInDomain(b *testing.B) {
	err := errNegativeMaxAge.WithContextAny("max_age", -5)

	b.ResetTimer()

	for b.Loop() {
		_ = InDomain(err, Domain("cors"))
	}
}
