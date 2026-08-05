package httpspec

import (
	"net/http"
	"strconv"
	"testing"
)

// newCORSAwareHandler returns a handler that sets the standard CORS headers
// for a specific origin. Used to test CORS specs pass against a properly
// configured handler.
func newCORSAwareHandler(origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

// newRateLimitedHandler returns a handler that rejects the first 2 requests
// from any given remote address with 429 + Retry-After, then allows the rest.
// Used to test rate-limit specs.
func newRateLimitedHandler() http.Handler {
	counts := make(map[string]int)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counts[r.RemoteAddr]++

		if counts[r.RemoteAddr] <= 2 {
			w.Header().Set("Retry-After", "60")
			//nolint:canonicalheader // industry-standard spelling
			w.Header().Set("X-RateLimit-Limit", "2")
			//nolint:canonicalheader // industry-standard spelling
			w.Header().Set("X-RateLimit-Remaining", "0")
			//nolint:canonicalheader // industry-standard spelling
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(int64(60), 10))
			w.WriteHeader(http.StatusTooManyRequests)

			return
		}

		//nolint:canonicalheader // industry-standard spelling
		w.Header().Set("X-RateLimit-Limit", "2")
		//nolint:canonicalheader // industry-standard spelling
		w.Header().Set("X-RateLimit-Remaining", "1")
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORSSpecs_PassWithProperHeaders(t *testing.T) {
	t.Parallel()

	handler := newCORSAwareHandler("https://example.com")

	for _, spec := range CORSSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()
			result := spec.Check(handler)
			if !result.OK {
				t.Errorf("spec %q failed: %s", spec.Name, result.Message)
			}
		})
	}
}

func TestCORSSpecs_FailWithoutAllowOrigin(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	found := false

	for _, spec := range CORSSpecs() {
		if spec.Name == SpecNameCORSAllowOrigin {
			found = true
			result := spec.Check(handler)
			if result.OK {
				t.Errorf("expected CORS spec to fail without Allow-Origin")
			}
		}
	}

	if !found {
		t.Fatal("SpecNameCORSAllowOrigin not found in CORSSpecs")
	}
}

func TestCORSSpecs_FailWildcardWithCredentials(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	})

	for _, spec := range CORSSpecs() {
		if spec.Name != SpecNameCORSWildcardNoCredentials {
			continue
		}

		result := spec.Check(handler)
		if result.OK {
			t.Errorf("expected wildcard+credentials spec to fail")
		}

		return
	}

	t.Fatal("SpecNameCORSWildcardNoCredentials not found in CORSSpecs")
}

func TestCORSSpecs_FailWithoutVaryOnDynamicOrigin(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://dynamic.example.com")
		// missing Vary: Origin
		w.WriteHeader(http.StatusOK)
	})

	for _, spec := range CORSSpecs() {
		if spec.Name != SpecNameCORSVaryOrigin {
			continue
		}

		result := spec.Check(handler)
		if result.OK {
			t.Errorf("expected Vary: Origin spec to fail without Vary header")
		}

		return
	}

	t.Fatal("SpecNameCORSVaryOrigin not found in CORSSpecs")
}

func TestRateLimitSpecs_PassWith429AndRetryAfter(t *testing.T) {
	t.Parallel()

	handler := newRateLimitedHandler()

	for _, spec := range RateLimitSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()
			result := spec.Check(handler)
			if !result.OK {
				t.Errorf("spec %q failed: %s", spec.Name, result.Message)
			}
		})
	}
}

func TestRateLimitSpecs_PassWithoutRateLimiting(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, spec := range RateLimitSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()
			result := spec.Check(handler)
			if !result.OK {
				t.Errorf("rate-limit spec %q failed on unrestricted handler: %s",
					spec.Name, result.Message)
			}
		})
	}
}

func TestRateLimitSpecs_FailOnInvalidRetryAfter(t *testing.T) {
	t.Parallel()

	counts := make(map[string]int)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counts[r.RemoteAddr]++

		if counts[r.RemoteAddr] <= 1 {
			w.Header().Set("Retry-After", "not-a-number")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	for _, spec := range RateLimitSpecs() {
		if spec.Name != SpecNameRateLimitHeaderOnReject {
			continue
		}

		result := spec.Check(handler)
		if result.OK {
			t.Errorf("expected spec to fail with non-integer Retry-After")
		}

		return
	}

	t.Fatal("SpecNameRateLimitHeaderOnReject not found in RateLimitSpecs")
}

func TestVaryContainsToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		vary  string
		token string
		want  bool
	}{
		{"single token match", "Origin", "Origin", true},
		{"case-insensitive", "origin", "Origin", true},
		{"multi-token match", "Accept-Encoding, Origin", "Origin", true},
		{"multi-token match case-insensitive", "accept-encoding, ORIGIN", "Origin", true},
		{"whitespace padded", " Origin , Accept-Encoding", "Origin", true},
		{"missing", "Accept-Encoding", "Origin", false},
		{"empty", "", "Origin", false},
		{"partial", "X-Origin", "Origin", false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := varyContainsToken(testCase.vary, testCase.token)
			if got != testCase.want {
				t.Errorf("varyContainsToken(%q, %q) = %v, want %v",
					testCase.vary, testCase.token, got, testCase.want)
			}
		})
	}
}

func TestValidateNonNegativeInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		wantFail bool
	}{
		{"empty", "", false},
		{"zero", "0", false},
		{"positive", "42", false},
		{"negative", "-1", true},
		{"non-numeric", "abc", true},
		{"float", "1.5", true},
		{"trailing", "12abc", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			msg := validateNonNegativeInt("X-Test", testCase.value)
			gotFail := msg != ""

			if gotFail != testCase.wantFail {
				t.Errorf("validateNonNegativeInt(%q) fail=%v, want %v (msg=%q)",
					testCase.value, gotFail, testCase.wantFail, msg)
			}
		})
	}
}
