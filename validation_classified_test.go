package httputil

import (
	"crypto/tls"
	"errors"
	"net/http"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// The tests in this file assert the typed hierarchical error model for every
// config validator and runtime error path: each error matches its sentinel
// via errors.Is, implements Coded/Classified/Contextual, and reports the
// expected code, domain, and family.

func TestCORSValidationErrorsClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	cfg.MaxAge = -1
	assertValidationClassified(
		t,
		cfg.Validate(),
		errNegativeMaxAge,
		codeCorsMaxAgeNegative,
		errorfamily.Rejection,
	)

	cfg = DefaultCORSConfig()
	cfg.AllowCredentials = true
	cfg.AllowAllOrigins = true
	assertValidationClassified(
		t,
		cfg.Validate(),
		errCredentialsWithAllOrigins,
		codeCorsCredentialsWithAllOrigins,
		errorfamily.Rejection,
	)
}

func TestCORSErrorCarriesContext(t *testing.T) {
	t.Parallel()

	cfg := DefaultCORSConfig()
	cfg.MaxAge = -5

	contextual, ok := errors.AsType[errorfamily.Contextual](cfg.Validate())
	if !ok {
		t.Fatal("Validate() error does not implement Contextual")
	}

	if got := contextual.ErrorContext()["max_age"]; got != "-5" {
		t.Errorf("context max_age = %q, want %q", got, "-5")
	}
}

func TestServerValidationErrorsClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.ReadTimeout = -1
	assertValidationClassified(
		t,
		cfg.Validate(),
		errReadTimeoutNegative,
		codeServerReadTimeoutNegative,
		errorfamily.Rejection,
	)

	cfg = DefaultServerConfig()
	cfg.ReadHeaderTimeout = 10 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	assertValidationClassified(
		t,
		cfg.Validate(),
		errServerTimeoutOrdering,
		codeServerTimeoutOrdering,
		errorfamily.Rejection,
	)

	cfg = DefaultServerConfig()
	//nolint:gosec // intentionally insecure MinVersion to exercise validation
	cfg.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS10}
	assertValidationClassified(
		t,
		cfg.Validate(),
		errTLSMinVersionInsecure,
		codeServerTLSMinVersionInsecure,
		errorfamily.Rejection,
	)
}

func TestServerAddrEmptyClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = ""
	assertValidationClassified(
		t,
		cfg.Validate(),
		errServerAddrEmpty,
		codeServerAddrEmpty,
		errorfamily.Rejection,
	)
}

func TestServerShutdownFailedClassified(t *testing.T) {
	t.Parallel()

	cause := errTestBoom
	err := errServerShutdownFailed.WithCause(cause)

	assertValidationClassified(
		t,
		err,
		errServerShutdownFailed,
		codeServerShutdownFailed,
		errorfamily.Infrastructure,
	)

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestCompressionValidationErrorsClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	cfg.Level = 99
	assertValidationClassified(
		t,
		cfg.Validate(),
		errInvalidCompressionLevel,
		codeCompressionLevelInvalid,
		errorfamily.Rejection,
	)

	cfg = DefaultCompressionConfig()
	cfg.MinSize = -1
	assertValidationClassified(
		t,
		cfg.Validate(),
		errNegativeMinSize,
		codeCompressionMinSizeNeg,
		errorfamily.Rejection,
	)

	cfg = DefaultCompressionConfig()
	cfg.WriterFactories = nil
	assertValidationClassified(
		t,
		cfg.Validate(),
		errNoWriterFactory,
		codeCompressionNoFactory,
		errorfamily.Rejection,
	)
}

func TestQValueParseErrorsClassified(t *testing.T) {
	t.Parallel()

	_, err := parseQValue("")
	assertValidationClassified(
		t,
		err,
		errEmptyQValue,
		codeCompressionQValueEmpty,
		errorfamily.Rejection,
	)

	_, err = parseQValue("x")
	assertValidationClassified(
		t,
		err,
		errInvalidQInt,
		codeCompressionQValueInvalid,
		errorfamily.Rejection,
	)

	_, err = parseQValue("0.5x")
	assertValidationClassified(
		t,
		err,
		errTrailingQChars,
		codeCompressionQValueTrail,
		errorfamily.Rejection,
	)

	_, err = parseQValue("1.5")
	assertValidationClassified(
		t,
		err,
		errQValueTooLarge,
		codeCompressionQValueTooBig,
		errorfamily.Rejection,
	)
}

func TestKeyedRateLimiterValidationErrorsClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultKeyedRateLimiterConfig()
	cfg.Limit = 0
	assertValidationClassified(
		t,
		cfg.Validate(),
		errKeyedLimitZero,
		codeRatelimitKeyedLimitZero,
		errorfamily.Rejection,
	)

	cfg = DefaultKeyedRateLimiterConfig()
	cfg.Window = -1
	assertValidationClassified(
		t,
		cfg.Validate(),
		errKeyedWindowZero,
		codeRatelimitKeyedWindowZero,
		errorfamily.Rejection,
	)

	cfg = DefaultKeyedRateLimiterConfig()
	cfg.TTL = -1
	assertValidationClassified(
		t,
		cfg.Validate(),
		errKeyedTTLNegative,
		codeRatelimitKeyedTTLNegative,
		errorfamily.Rejection,
	)
}

func TestMaxBodySizeValidationErrorClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultMaxBodySizeConfig()
	cfg.MaxBytes = -1
	assertValidationClassified(
		t,
		cfg.Validate(),
		errMaxBodySizeNegative,
		codeMaxBodySizeNegative,
		errorfamily.Rejection,
	)
}

func TestRequestIDValidationErrorsClassified(t *testing.T) {
	t.Parallel()

	cfg := newRequestIDConfigForTest("X-Request-Id", "X-Request-Id")
	cfg.GenerateID = nil
	assertValidationClassified(
		t,
		cfg.Validate(),
		errNilGenerateID,
		codeRequestIDNilGenerateID,
		errorfamily.Rejection,
	)

	cfg = newRequestIDConfigForTest("", "X-Request-Id")
	assertValidationClassified(
		t,
		cfg.Validate(),
		errEmptyResponseHeader,
		codeRequestIDEmptyResponseHeader,
		errorfamily.Rejection,
	)

	cfg = newRequestIDConfigForTest("X-Request-Id", "")
	assertValidationClassified(
		t,
		cfg.Validate(),
		errEmptyIncomingHeader,
		codeRequestIDEmptyIncomingHeader,
		errorfamily.Rejection,
	)
}

func TestSecurityHeadersValidationErrorClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultSecurityHeadersConfig()
	cfg.FrameOptions = "MAYBE"
	assertValidationClassified(
		t,
		cfg.Validate(),
		errSecurityInvalidFrameOptions,
		codeSecurityInvalidFrameOptions,
		errorfamily.Rejection,
	)
}

func TestMetricsValidationErrorClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultMetricsConfig()
	cfg.Recorder = nil
	assertValidationClassified(
		t,
		cfg.Validate(),
		errNilMetricsRecorder,
		codeMetricsNilRecorder,
		errorfamily.Rejection,
	)
}

func TestNonceValidationErrorClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultNonceConfig()
	cfg.Size = 8
	assertValidationClassified(
		t,
		cfg.Validate(),
		errNonceTooSmall,
		codeNonceTooSmall,
		errorfamily.Rejection,
	)
}

func TestRateLimitValidationErrorsClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultRateLimitConfig()
	cfg.Limiter = nil
	assertValidationClassified(
		t,
		cfg.Validate(),
		errNilRateLimiter,
		codeRatelimitNilLimiter,
		errorfamily.Rejection,
	)

	cfg = DefaultRateLimitConfig()
	cfg.Limiter = &alwaysDenyLimiter{}
	cfg.Status = 42
	assertValidationClassified(
		t,
		cfg.Validate(),
		errInvalidStatus,
		codeRatelimitInvalidStatus,
		errorfamily.Rejection,
	)
}

func TestStackValidationErrorsClassified(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()
	mw := func(next http.Handler) http.Handler { return next }

	err := stack.Add(MiddlewareLogging, mw)
	if err != nil {
		t.Fatalf("first Add() err = %v, want nil", err)
	}

	err = stack.Add(MiddlewareLogging, mw)
	assertValidationClassified(
		t,
		err,
		errDuplicateMiddleware,
		codeStackDuplicateMiddleware,
		errorfamily.Rejection,
	)
}

func TestStackRecoveryNotFirstClassified(t *testing.T) {
	t.Parallel()

	stack := NewMiddlewareStack()
	mw := func(next http.Handler) http.Handler { return next }

	err := stack.Add(MiddlewareLogging, mw)
	if err != nil {
		t.Fatalf("first Add() err = %v, want nil", err)
	}

	err = stack.Add(MiddlewareRecovery, mw)
	if err != nil {
		t.Fatalf("second Add() err = %v, want nil", err)
	}

	assertValidationClassified(
		t,
		stack.Validate(),
		errRecoveryNotFirst,
		codeStackRecoveryNotFirst,
		errorfamily.Rejection,
	)
}

func TestDecompressionConfigValidationErrorClassified(t *testing.T) {
	t.Parallel()

	cfg := DefaultDecompressionConfig()
	cfg.MaxDecompressionSize = -1
	assertValidationClassified(
		t,
		cfg.Validate(),
		errMaxDecompressionSizeNegative,
		codeDecompressionSizeNegative,
		errorfamily.Rejection,
	)
}

func TestDecompressionSizeExceededClassified(t *testing.T) {
	t.Parallel()

	assertValidationClassified(
		t,
		errDecompressionSizeExceeded,
		errDecompressionSizeExceeded,
		codeDecompressionSizeExceeded,
		errorfamily.Rejection,
	)
}

func TestDecompressionReadFailedClassified(t *testing.T) {
	t.Parallel()

	cause := errTestBoom
	err := codeDecompressionReadFailed.WrapCorruption(cause, "decompression read failed")

	assertValidationClassified(
		t,
		err,
		codeDecompressionReadFailed.Corruption("decompression read failed"),
		codeDecompressionReadFailed,
		errorfamily.Corruption,
	)

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestDecompressionCloseFailedClassified(t *testing.T) {
	t.Parallel()

	cause := errTestBoom
	err := codeDecompressionCloseFailed.WrapTransient(cause, "decompression close failed")

	assertValidationClassified(
		t,
		err,
		codeDecompressionCloseFailed.Transient("decompression close failed"),
		codeDecompressionCloseFailed,
		errorfamily.Transient,
	)

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestCSRFValidationErrorsClassifiedWithCauseChain(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{}
	cfg.SameSite = 4 // http.SameSiteNoneMode
	cfg.Secure = false

	err := cfg.Validate()

	assertValidationClassified(
		t,
		err,
		codeCSRFSameSiteInsecure.Infrastructure("SameSite=None requires Secure=true"),
		codeCSRFSameSiteInsecure,
		errorfamily.Infrastructure,
	)

	if !errors.Is(err, ErrCSRFConfig) {
		t.Errorf("errors.Is(err, ErrCSRFConfig) = false, want true")
	}

	if !InDomain(err, codeCSRFSameSiteInsecure.Domain()) {
		t.Errorf("InDomain(err, csrf) = false, want true")
	}
}

func TestCSRFUnsafeOriginCarriesContext(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{}
	cfg.Secure = true
	cfg.TrustedOrigins = []string{"*"}

	err := cfg.Validate()

	if !errors.Is(err, codeCSRFUnsafeOrigin.Infrastructure("x")) {
		t.Errorf("error does not match csrf_unsafe_origin code")
	}

	contextual, ok := errors.AsType[errorfamily.Contextual](err)
	if !ok {
		t.Fatal("error does not implement Contextual")
	}

	if got := contextual.ErrorContext()["origin"]; got != "*" {
		t.Errorf("context origin = %q, want %q", got, "*")
	}
}

func TestCSRFInvalidCIDRClassified(t *testing.T) {
	t.Parallel()

	cfg := CSRFConfig{}
	cfg.Secure = true
	cfg.TrustedProxies = []string{"not-a-cidr/"}

	err := cfg.Validate()

	assertValidationClassified(
		t,
		err,
		codeCSRFInvalidCIDR.Infrastructure("TrustedProxies contains invalid CIDR"),
		codeCSRFInvalidCIDR,
		errorfamily.Infrastructure,
	)

	if !errors.Is(err, ErrCSRFConfig) {
		t.Errorf("errors.Is(err, ErrCSRFConfig) = false, want true")
	}
}
