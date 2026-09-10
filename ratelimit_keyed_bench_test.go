package httputil

import (
	"strconv"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// BenchmarkKeyedRateLimiter_MaxKeysChurn measures the true slow path under
// sustained MaxKeys pressure: every iteration inserts a fresh key, so each
// insert at capacity pays the heap-pop eviction plus push.
func BenchmarkKeyedRateLimiter_MaxKeysChurn(b *testing.B) {
	const maxKeys = 1024

	p := newPerKeyLimiter(rate.Limit(1e9), 1_000_000_000, nil, "0", time.Hour, maxKeys)

	i := 0

	for b.Loop() {
		i++
		_ = p.limiter("key-" + strconv.Itoa(i))
	}
}
