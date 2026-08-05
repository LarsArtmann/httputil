package httpspec

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
)

// Rate-limit header conventions: the de-facto industry spelling is
// "X-RateLimit-*" (capital L). Go canonicalizes to "X-Ratelimit-*" but
// servers and clients must accept both spellings — http.Header.Get is
// case-insensitive, and the canonical form has no semantic meaning on the
// wire. We test against the conventional spelling because that's what
// real services emit (GitHub, Twitter, Stripe, etc.). The non-canonical
// literal is intentional: it's the spelling that real-world handlers use.
const (
	headerXRateLimitLimit     = "X-RateLimit-Limit"
	headerXRateLimitRemaining = "X-RateLimit-Remaining"
	headerXRateLimitReset     = "X-RateLimit-Reset"
)

// CORS spec names identify each CORS behavior check for use with [SkipSpec].
const (
	SpecNameCORSAllowOrigin           = "cross-origin responses should include Access-Control-Allow-Origin"
	SpecNameCORSAllowCredentials      = "Access-Control-Allow-Credentials should reflect credential policy"
	SpecNameCORSVaryOrigin            = "responses should set Vary: Origin when origin is dynamic"
	SpecNameCORSWildcardNoCredentials = "Access-Control-Allow-Origin: * must not be combined with credentials"
)

// Rate-limit spec names identify each rate-limit behavior check for use with [SkipSpec].
const (
	SpecNameRateLimitRetryAfter         = "rate-limited responses should include a Retry-After header"
	SpecNameRateLimitHeaderOnReject     = "rejected responses should set Retry-After as a non-negative integer"
	SpecNameRateLimitHintHeadersOnAllow = "successful responses may include X-RateLimit-* hints"
)

// CORSSpecs returns CORS behavior specs that can be composed into a spec run
// via [WithExtraSpecs]. These specs verify the CORS headers set by a handler
// conform to spec — they assume the handler intends to serve cross-origin
// requests. For handlers that do not serve cross-origin requests, omit these.
//
// The returned specs assume the handler is intended to be a CORS-aware service
// (e.g. wrapped with cors middleware). Use [SkipSpec] for individual specs
// that are not applicable to your deployment.
func CORSSpecs() []Spec {
	return []Spec{
		{
			Name:     SpecNameCORSAllowOrigin,
			Category: CategoryHeaders,
			Check:    corsAllowOriginCheck(),
		},
		{
			Name:     SpecNameCORSAllowCredentials,
			Category: CategorySecurity,
			Check:    corsAllowCredentialsCheck(),
		},
		{
			Name:     SpecNameCORSVaryOrigin,
			Category: CategoryHeaders,
			Check:    corsVaryOriginCheck(),
		},
		{
			Name:     SpecNameCORSWildcardNoCredentials,
			Category: CategorySecurity,
			Check:    corsWildcardNoCredentialsCheck(),
		},
	}
}

// RateLimitSpecs returns rate-limit behavior specs that can be composed into a
// spec run via [WithExtraSpecs]. These specs verify the rate-limit headers set
// by a handler conform to RFC 6585 (Retry-After) and the de-facto
// X-RateLimit-* conventions.
//
// The specs assume the handler is intended to enforce rate limits (e.g.
// wrapped with a token-bucket middleware). For handlers without rate limiting,
// omit these.
func RateLimitSpecs() []Spec {
	return []Spec{
		{
			Name:     SpecNameRateLimitRetryAfter,
			Category: CategoryHeaders,
			Check:    rateLimitRetryAfterCheck(),
		},
		{
			Name:     SpecNameRateLimitHeaderOnReject,
			Category: CategoryHeaders,
			Check:    rateLimitHeaderOnRejectCheck(),
		},
		{
			Name:     SpecNameRateLimitHintHeadersOnAllow,
			Category: CategoryHeaders,
			Check:    rateLimitHintHeadersOnAllowCheck(),
		},
	}
}

// corsAllowOriginCheck verifies that successful responses carry an
// Access-Control-Allow-Origin header when the request includes an Origin. The
// header is the linchpin of the CORS protocol — its absence means browsers
// will reject the response client-side even if other CORS headers are correct.
func corsAllowOriginCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, "/")
		req.Header.Set("Origin", "https://example.com")

		rec := serve(handler, req)

		acao := rec.Header().Get("Access-Control-Allow-Origin")
		if acao == "" {
			return Fail(
				"cross-origin GET / with Origin: https://example.com returned no Access-Control-Allow-Origin header " +
					"(browsers will block the response client-side)",
			)
		}

		return Pass()
	}
}

// corsAllowCredentialsCheck verifies that Access-Control-Allow-Credentials is
// set to a valid boolean string ("true" or "false"). Browsers require an exact
// match — anything else is rejected.
func corsAllowCredentialsCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, "/")
		req.Header.Set("Origin", "https://example.com")

		rec := serve(handler, req)

		acac := rec.Header().Get("Access-Control-Allow-Credentials")
		if acac == "" {
			return Pass() // header not set: only validate when present
		}

		if acac != "true" && acac != "false" {
			return Fail(
				"Access-Control-Allow-Credentials = %q, must be exactly \"true\" or \"false\" "+
					"(browsers reject other values per the CORS spec)",
				acac,
			)
		}

		return Pass()
	}
}

// corsVaryOriginCheck verifies that responses set Vary: Origin when the
// Access-Control-Allow-Origin header value depends on the request origin.
// Without Vary: Origin, intermediate caches can serve a cached CORS response
// from one origin to a request from another, leaking credentials or denying
// access incorrectly.
func corsVaryOriginCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, "/")
		req.Header.Set("Origin", "https://example.com")

		rec := serve(handler, req)

		acao := rec.Header().Get("Access-Control-Allow-Origin")
		if acao == "" || acao == "*" {
			return Pass() // static origin: no Vary needed
		}

		vary := rec.Header().Get("Vary")
		if !varyContainsToken(vary, "Origin") {
			return Fail(
				"Access-Control-Allow-Origin = %q (dynamic), but Vary header = %q "+
					"does not include \"Origin\"; caches may serve this response to other origins",
				acao, vary,
			)
		}

		return Pass()
	}
}

// corsWildcardNoCredentialsCheck verifies that Access-Control-Allow-Origin: *
// is never combined with Access-Control-Allow-Credentials: true. The CORS spec
// forbids this combination — browsers reject it outright.
func corsWildcardNoCredentialsCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, "/")
		req.Header.Set("Origin", "https://example.com")

		rec := serve(handler, req)

		acao := rec.Header().Get("Access-Control-Allow-Origin")
		acac := rec.Header().Get("Access-Control-Allow-Credentials")

		if acao == "*" && strings.EqualFold(acac, "true") {
			return Fail(
				"Access-Control-Allow-Origin: * combined with Access-Control-Allow-Credentials: true " +
					"is forbidden by the CORS spec — browsers reject this combination",
			)
		}

		return Pass()
	}
}

// rateLimitRetryAfterCheck verifies that 429 Too Many Requests responses
// include a Retry-After header (RFC 6585 §4). Without Retry-After, well-behaved
// clients cannot back off and will keep hammering the endpoint.
func rateLimitRetryAfterCheck() Check {
	return func(handler http.Handler) Result {
		// Repeated requests with a small burst should eventually trigger 429.
		// If the handler does not rate-limit at all, this check passes trivially.
		rec, ok := firstTooManyRequests(handler, "192.0.2.1:1234")
		if !ok {
			return Pass() // no rate limiting detected: skip
		}

		if rec.Header().Get("Retry-After") == "" {
			return Fail(
				"429 Too Many Requests returned no Retry-After header " +
					"(RFC 6585 §4 requires it so clients can back off correctly)",
			)
		}

		return Pass()
	}
}

// rateLimitHeaderOnRejectCheck verifies that Retry-After on 429 responses is a
// non-negative integer (seconds). Non-integer values are valid per RFC 7231
// but most clients only parse integer seconds; a non-integer value risks
// confusing clients.
func rateLimitHeaderOnRejectCheck() Check {
	return func(handler http.Handler) Result {
		rec, ok := firstTooManyRequests(handler, "192.0.2.2:1234")
		if !ok {
			return Pass()
		}

		retryAfter := rec.Header().Get("Retry-After")
		if retryAfter == "" {
			return Pass() // already covered by rateLimitRetryAfterCheck
		}

		secs, err := strconv.Atoi(retryAfter)
		if err != nil {
			return Fail(
				"Retry-After = %q is not a non-negative integer; most clients only parse integer seconds",
				retryAfter,
			)
		}

		if secs < 0 {
			return Fail(
				"Retry-After = %d is negative; clients interpret negative values as immediate retry",
				secs,
			)
		}

		return Pass()
	}
}

// rateLimitHintHeadersOnAllowCheck verifies that when rate-limit hint headers
// are present, they are well-formed non-negative integers. The check is
// permissive: missing headers are OK (some implementations don't expose hints).
func rateLimitHintHeadersOnAllowCheck() Check {
	return func(handler http.Handler) Result {
		req := mustRequest(http.MethodGet, "/")
		req.RemoteAddr = "192.0.2.3:1234"
		rec := serve(handler, req)

		limit := rec.Header().Get(headerXRateLimitLimit)
		remaining := rec.Header().Get(headerXRateLimitRemaining)
		reset := rec.Header().Get(headerXRateLimitReset)

		if limit == "" && remaining == "" && reset == "" {
			return Pass() // no hints provided: skip
		}

		if msg := validateNonNegativeInt(headerXRateLimitLimit, limit); msg != "" {
			return Fail("%s", msg)
		}

		if msg := validateNonNegativeInt(headerXRateLimitRemaining, remaining); msg != "" {
			return Fail("%s", msg)
		}

		if msg := validateNonNegativeInt(headerXRateLimitReset, reset); msg != "" {
			return Fail("%s", msg)
		}

		return Pass()
	}
}

// firstTooManyRequests sends up to 100 requests from the given remote address
// and returns the first 429 response, or ok=false if no 429 is produced.
// This is a test fixture for rate-limit checks — we don't know if a handler
// actually rate-limits, so we probe first.
func firstTooManyRequests(
	handler http.Handler,
	remoteAddr string,
) (*httptest.ResponseRecorder, bool) {
	for range 100 {
		req := mustRequest(http.MethodGet, "/")
		req.RemoteAddr = remoteAddr
		rec := serve(handler, req)

		if rec.Code == http.StatusTooManyRequests {
			return rec, true
		}
	}

	return nil, false
}

// validateNonNegativeInt returns an error message if value is not a valid
// non-negative integer, or "" if it is valid (or empty).
func validateNonNegativeInt(name, value string) string {
	if value == "" {
		return ""
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return name + " = " + strconv.Quote(value) + " is not a valid non-negative integer"
	}

	return ""
}

// varyContainsToken reports whether the Vary header value contains the given
// token in a case-insensitive comparison (Vary tokens are case-insensitive per
// RFC 7231 §7.1.4).
func varyContainsToken(vary, token string) bool {
	for t := range strings.SplitSeq(vary, ",") {
		if strings.EqualFold(strings.TrimSpace(t), token) {
			return true
		}
	}

	return false
}
