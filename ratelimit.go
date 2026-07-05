package httputil

import (
	"errors"
	"math"
	"net/http"
	"sync"
	"time"
)

var (
	errNilRateLimiter = errors.New("rate limit config: Limiter must not be nil")
	errInvalidRate    = errors.New("rate must be greater than zero")
	errInvalidBurst   = errors.New("burst must be greater than zero")
)

const (
	defaultRateLimitStatus = http.StatusTooManyRequests
	tokenCost              = 1.0
)

// RateLimiter decides whether a request should be allowed based on a key.
// Implementations must be safe for concurrent use.
type RateLimiter interface {
	// Allow returns true if the request identified by key should be allowed,
	// consuming one token from the bucket. Returns false if the rate limit
	// has been exceeded.
	Allow(key string) bool
}

// TokenBucketLimiter is a simple in-memory rate limiter using a token bucket
// per key. Tokens refill at a fixed rate up to the burst capacity.
type TokenBucketLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    float64 // tokens per second
	burst   float64 // maximum tokens
	now     func() time.Time
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

// NewTokenBucketLimiter creates a RateLimiter that allows requests at the
// given rate (tokens per second) with a burst capacity. Each unique key
// gets its own bucket. Returns an error if rate or burst is not positive.
func NewTokenBucketLimiter(rate, burst float64) (*TokenBucketLimiter, error) {
	if rate <= 0 {
		return nil, errInvalidRate
	}

	if burst <= 0 {
		return nil, errInvalidBurst
	}

	return &TokenBucketLimiter{
		mu:      sync.Mutex{},
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   burst,
		now:     time.Now,
	}, nil
}

// Allow returns true if the key has tokens remaining, consuming one token.
func (l *TokenBucketLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	now := l.now()

	if !ok {
		bucket = &tokenBucket{tokens: l.burst, lastRefill: now}
		l.buckets[key] = bucket
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = math.Min(l.burst, bucket.tokens+elapsed*l.rate)
	bucket.lastRefill = now

	if bucket.tokens >= tokenCost {
		bucket.tokens -= tokenCost

		return true
	}

	return false
}

// RateLimitConfig holds configuration for the rate limiting middleware.
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

	return nil
}

// RateLimit returns middleware that enforces rate limiting using the configured
// [RateLimiter]. Requests exceeding the limit receive the configured denial
// response (default: 429 Too Many Requests).
func RateLimit(cfg RateLimitConfig) Middleware {
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
