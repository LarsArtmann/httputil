package httputil

import (
	"slices"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	etag "github.com/larsartmann/go-etag"
)

// allHTTputilErrorCodes is the authoritative list of every error code the
// package can produce. The completeness test asserts a registered message
// template exists for each. When you add a code, add it here and register a
// template in errors.go.
var allHTTputilErrorCodes = []Code{
	codeWriteFailed,
	codeHijackUnsupported,
	codeHijackFailed,
	codeCompressWriteFailed,
	codeCompressionPoolTypeUnexpected,

	codeCorsCredentialsWithAllOrigins,
	codeCorsMaxAgeNegative,

	codeServerAddrEmpty,
	codeServerReadTimeoutNegative,
	codeServerReadHeaderTimeoutNegative,
	codeServerWriteTimeoutNegative,
	codeServerIdleTimeoutNegative,
	codeServerShutdownTimeoutNegative,
	codeServerTimeoutOrdering,
	codeServerTLSMinVersionInsecure,
	codeServerShutdownFailed,

	codeCompressionLevelInvalid,
	codeCompressionMinSizeNeg,
	codeCompressionNoFactory,
	codeCompressionQValueEmpty,
	codeCompressionQValueInvalid,
	codeCompressionQValueTrail,
	codeCompressionQValueTooBig,

	codeRatelimitKeyedLimitZero,
	codeRatelimitKeyedWindowZero,
	codeRatelimitKeyedTTLNegative,
	codeRatelimitNilLimiter,
	codeRatelimitInvalidRate,
	codeRatelimitInvalidBurst,
	codeRatelimitInvalidStatus,

	codeMaxBodySizeNegative,

	codeRequestIDNilGenerateID,
	codeRequestIDEmptyResponseHeader,
	codeRequestIDEmptyIncomingHeader,

	codeSecurityInvalidFrameOptions,

	codeMetricsNilRecorder,

	codeNonceTooSmall,

	codeStackDuplicateMiddleware,
	codeStackRecoveryNotFirst,

	codeDecompressionSizeNegative,
	codeDecompressionSizeExceeded,
	codeDecompressionReadFailed,
	codeDecompressionCloseFailed,
}

// legacyErrorCodes are pre-taxonomy codes kept for backward compatibility.
// They use the historical underscore spelling (no domain prefix) and are
// exempt from the domain-prefix assertion but still require templates.
var legacyErrorCodes = []string{
	string(codeCSRFInvalid),
	string(codeCSRFConfig),
	string(codeCSRFSameSiteInsecure),
	string(codeCSRFUnsafeOrigin),
	string(codeCSRFUnsafeProxy),
	string(codeCSRFInvalidCIDR),
}

func TestEveryErrorCodeHasATemplate(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	for _, code := range slices.Concat(allHTTputilErrorCodes, codesOf(legacyErrorCodes)) {
		tmpl, ok := errorfamily.TemplateForCode(string(code))
		if !ok {
			t.Errorf("no message template registered for code %q", string(code))

			continue
		}

		if tmpl.What == "" || tmpl.Why == "" || tmpl.Fix == "" || tmpl.WayOut == "" {
			t.Errorf(
				"template for code %q is incomplete: what=%q why=%q fix=%q way_out=%q",
				string(code), tmpl.What, tmpl.Why, tmpl.Fix, tmpl.WayOut,
			)
		}
	}
}

func TestEveryCodeUsesADomainPrefix(t *testing.T) {
	t.Parallel()

	for _, code := range allHTTputilErrorCodes {
		if code.Domain() == Domain("") || code.Domain() == Domain(code) {
			t.Errorf("code %q has no domain prefix", string(code))
		}
	}
}

func TestCSRFExportedCodesHaveTemplates(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	for _, code := range []string{string(codeCSRFInvalid), string(codeCSRFConfig)} {
		if _, ok := errorfamily.TemplateForCode(code); !ok {
			t.Errorf("no message template registered for exported CSRF code %q", code)
		}
	}
}

// codesOf converts string codes to typed Codes for shared iteration.
func codesOf(codes []string) []Code {
	//nolint:makezero // pre-allocated with known length, not append
	typed := make([]Code, len(codes))
	for i, code := range codes {
		typed[i] = Code(code)
	}

	return typed
}

func TestETagCodesHaveTemplates(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	for _, code := range []string{
		etag.ErrCodeETagWriteFailed,
		etag.ErrCodeInvalidConfig,
		etag.ErrCodeHashWriteFailed,
	} {
		if _, ok := errorfamily.TemplateForCode(code); !ok {
			t.Errorf("no message template registered for ETag code %q", code)
		}
	}
}
