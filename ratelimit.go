package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	errNilRateLimiter = errors.New("rate limit config: Limiter must not be nil")
	errInvalidRate    = errors.New("rate must be greater than zero")
	errInvalidBurst   = errors.New("burst must be greater than zero")
	errInvalidStatus  = errors.New(
		"RateLimitConfig.Status must be a valid HTTP status code (100-599) or zero for default",
	)
)

const (
	defaultRateLimitStatus = http.StatusTooManyRequests
)

// RateLimiter decides whether a request should be allowed based on a key.
// Implementations must be safe for concurrent use.
//
// Deprecated: Use [KeyedRateLimiter], which adds min-heap eviction, a MaxKeys
// cap, and a monitoring API. This interface will be removed in a future release.
type RateLimiter interface {
	// Allow returns true if the request identified by key should be allowed,
	// consuming one token from the bucket. Returns false if the rate limit
	// has been exceeded.
	Allow(key string) bool
}

// rateBucket wraps a per-key rate.Limiter together with the last access
// time needed for idle-bucket eviction. The zero value is not usable.
type rateBucket struct {
	lim        *rate.Limiter
	lastAccess time.Time
}

// TokenBucketLimiter is an in-memory rate limiter using a token bucket per
// key, backed by golang.org/x/time/rate. Tokens refill at a fixed rate up to
// the burst capacity.
//
// Set EvictionTTL to a non-zero duration to enable lazy eviction of idle
// buckets — buckets that have not been accessed within EvictionTTL are
// removed on the next Allow call that triggers a sweep. Zero (the default)
// disables eviction, preserving the original unbounded-growth behavior.
//
// Deprecated: Use [KeyedRateLimiter], which provides O(log n) min-heap eviction,
// a MaxKeys cap, and a monitoring API.
type TokenBucketLimiter struct {
	mu sync.Mutex
	// EvictionTTL controls idle-bucket eviction. When non-zero, buckets not
	// accessed for this duration are evicted lazily on Allow calls.
	EvictionTTL time.Duration

	buckets   map[string]*rateBucket
	rate      rate.Limit
	burst     int
	now       func() time.Time
	lastSweep time.Time
}

// NewTokenBucketLimiter creates a RateLimiter that allows requests at the
// given rate (tokens per second) with a burst capacity. Each unique key gets
// its own bucket. Returns an error if rate or burst is not positive.
//
// Deprecated: Use [NewKeyedRateLimiter], which provides min-heap eviction and a
// MaxKeys cap.
func NewTokenBucketLimiter(rateValue float64, burst int) (*TokenBucketLimiter, error) {
	if rateValue <= 0 {
		return nil, errInvalidRate
	}

	if burst <= 0 {
		return nil, errInvalidBurst
	}

	return &TokenBucketLimiter{
		mu:          sync.Mutex{},
		EvictionTTL: 0,
		buckets:     make(map[string]*rateBucket),
		rate:        rate.Limit(rateValue),
		burst:       burst,
		now:         time.Now,
		lastSweep:   time.Time{},
	}, nil
}

// Allow returns true if the key has tokens remaining, consuming one token.
func (l *TokenBucketLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	if l.EvictionTTL > 0 && now.Sub(l.lastSweep) >= l.EvictionTTL {
		l.sweep(now)
	}

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &rateBucket{lim: rate.NewLimiter(l.rate, l.burst), lastAccess: now}
		l.buckets[key] = bucket
	}

	bucket.lastAccess = now

	return bucket.lim.AllowN(now, 1)
}

// sweep removes buckets that have been idle for longer than EvictionTTL.
// Must be called with the mutex held.
func (l *TokenBucketLimiter) sweep(now time.Time) {
	l.lastSweep = now

	for key, bucket := range l.buckets {
		if now.Sub(bucket.lastAccess) > l.EvictionTTL {
			delete(l.buckets, key)
		}
	}
}

// RateLimitConfig holds configuration for the rate limiting middleware.
//
// Deprecated: Use [KeyedRateLimiterConfig] with [KeyedRateLimiterMiddleware].
type RateLimitConfig struct {
	// Limiter decides whether to allow each request. Required.
	Limiter RateLimiter

	// KeyFunc extracts the rate-limiting key from the request (e.g., client IP).
	// If nil, the remote address is used.
	KeyFunc func(r *http.Request) string

	// Status is the HTTP status code returned when rate limited.
	// Defaults to 429 Too Many Requests. Ignored when OnDenied is set.
	Status int

	// OnDenied is called when the rate limiter rejects a request. If nil,
	// the middleware writes a simple text response with the configured Status.
	OnDenied http.HandlerFunc
}

// DefaultRateLimitConfig returns a config with sensible defaults. The caller
// must set Limiter before use.
//
// Deprecated: Use [DefaultKeyedRateLimiterConfig].
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limiter:  nil,
		KeyFunc:  nil,
		Status:   defaultRateLimitStatus,
		OnDenied: nil,
	}
}

// Validate checks the RateLimitConfig for invalid values.
func (c RateLimitConfig) Validate() error {
	if c.Limiter == nil {
		return errNilRateLimiter
	}

	if c.Status != 0 && (c.Status < 100 || c.Status > 599) {
		return fmt.Errorf("%w: got %d", errInvalidStatus, c.Status)
	}

	return nil
}

// RateLimit returns middleware that enforces rate limiting using the configured
// [RateLimiter]. Requests exceeding the limit receive the configured denial
// response (default: 429 Too Many Requests).
//
// Deprecated: Use [KeyedRateLimiterMiddleware], which provides min-heap eviction,
// a MaxKeys cap, Retry-After headers, and a monitoring API.
func RateLimit(cfg RateLimitConfig) Middleware {
	validateConfig("RateLimitConfig", cfg.Validate())

	status := cfg.Status
	if status == 0 {
		status = defaultRateLimitStatus
	}

	keyFunc := cfg.KeyFunc
	if keyFunc == nil {
		keyFunc = func(r *http.Request) string {
			return r.RemoteAddr
		}
	}

	onDenied := cfg.OnDenied
	if onDenied == nil {
		onDenied = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)

			// Status already committed; write failure is unreportable.
			_, _ = w.Write([]byte("rate limit exceeded"))
		})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			if !cfg.Limiter.Allow(key) {
				onDenied.ServeHTTP(w, r)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
