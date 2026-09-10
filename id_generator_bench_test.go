package httputil

import (
	"crypto/rand"
	"testing"
)

// BenchmarkGenerateTimeOrderedID measures the amortized cost of the time-
// ordered ID generator, including the periodic crypto/rand refill path.
func BenchmarkGenerateTimeOrderedID(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = generateTimeOrderedID()
	}
}

// BenchmarkGenerateTimeOrderedIDParallel measures contention on the slot
// counter, the published-generation pointer, and the per-second counter under
// concurrent generation.
func BenchmarkGenerateTimeOrderedIDParallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = generateTimeOrderedID()
		}
	})
}

// BenchmarkIDGeneratorRefillSwap measures one full generation refill:
// a 2 KiB allocation plus a crypto/rand read, published atomically. The
// ns/ID-amortized metric divides the per-refill cost by the number of IDs a
// generation serves.
func BenchmarkIDGeneratorRefillSwap(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		target := uint64(1)

		if cur := randomState.Load(); cur != nil {
			target = cur.gen + 1
		}

		refillRandomBuffer(target)
	}

	b.ReportMetric(
		float64(b.Elapsed().Nanoseconds())/float64(b.N*randBufferIDs),
		"ns/ID-amortized",
	)
}

// BenchmarkIDGeneratorRefillRawRandRead is the baseline for the swap refill:
// the bare crypto/rand read into a reused 2 KiB buffer, no allocation and no
// publication.
func BenchmarkIDGeneratorRefillRawRandRead(b *testing.B) {
	buf := make([]byte, randBufferLen)

	b.ReportAllocs()

	for b.Loop() {
		if _, err := rand.Read(buf); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(
		float64(b.Elapsed().Nanoseconds())/float64(b.N*randBufferIDs),
		"ns/ID-amortized",
	)
}
