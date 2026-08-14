package httputil

import (
	"errors"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Code is a machine-readable error code in the httputil error taxonomy,
// e.g. "cors.max_age_negative" or "http.write_failed". Its prefix before
// the first dot is the Domain, identifying the failing component; the full
// code identifies the specific failure.
type Code string

// Domain identifies the failing component of an error code: the prefix
// before the first dot (e.g. "cors" for "cors.max_age_negative"). Domains
// are by component, not by lifecycle — the errorfamily Family encodes
// lifecycle (Rejection, Transient, ...), the Domain answers "which
// middleware failed".
type Domain string

// Domain returns the component prefix of the code: everything before the
// first dot, or the whole code when it contains no dot.
func (c Code) Domain() Domain {
	if idx := strings.IndexByte(string(c), '.'); idx >= 0 {
		return Domain(c[:idx])
	}

	return Domain(c)
}

// The constructor methods below each return a fresh *errorfamily.Error for
// the code, so package-level sentinels defined with them can be shared
// safely: WithContext, WithCause, and friends clone rather than mutate.

// Rejection returns a fresh Rejection-family error with this code.
// Rejection is the family for invalid configuration and unacceptable
// requests: retrying without changing the input cannot succeed.
func (c Code) Rejection(message string) *errorfamily.Error {
	return errorfamily.NewRejection(string(c), message)
}

// Conflict returns a fresh Conflict-family error with this code.
func (c Code) Conflict(message string) *errorfamily.Error {
	return errorfamily.NewConflict(string(c), message)
}

// Transient returns a fresh Transient-family error with this code.
func (c Code) Transient(message string) *errorfamily.Error {
	return errorfamily.NewTransient(string(c), message)
}

// Corruption returns a fresh Corruption-family error with this code.
func (c Code) Corruption(message string) *errorfamily.Error {
	return errorfamily.NewCorruption(string(c), message)
}

// Infrastructure returns a fresh Infrastructure-family error with this code.
func (c Code) Infrastructure(message string) *errorfamily.Error {
	return errorfamily.NewInfrastructure(string(c), message)
}

// Orchestration returns a fresh Orchestration-family error with this code.
func (c Code) Orchestration(message string) *errorfamily.Error {
	return errorfamily.NewOrchestration(string(c), message)
}

// The Wrap methods mirror the constructor methods for wrapping a cause.

// WrapRejection wraps cause in a fresh Rejection-family error with this code.
func (c Code) WrapRejection(cause error, message string) *errorfamily.Error {
	return errorfamily.WrapRejection(cause, string(c), message)
}

// WrapTransient wraps cause in a fresh Transient-family error with this code.
func (c Code) WrapTransient(cause error, message string) *errorfamily.Error {
	return errorfamily.WrapTransient(cause, string(c), message)
}

// WrapCorruption wraps cause in a fresh Corruption-family error with this code.
func (c Code) WrapCorruption(cause error, message string) *errorfamily.Error {
	return errorfamily.WrapCorruption(cause, string(c), message)
}

// WrapInfrastructure wraps cause in a fresh Infrastructure-family error with
// this code.
func (c Code) WrapInfrastructure(cause error, message string) *errorfamily.Error {
	return errorfamily.WrapInfrastructure(cause, string(c), message)
}

// DomainOf returns the error-code domain of err: the component prefix of
// its machine-readable code. It reports false when err carries no code
// (i.e. implements neither errorfamily.Coded nor a compatible ErrorCode
// method anywhere in its chain).
func DomainOf(err error) (Domain, bool) {
	coded, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		return "", false
	}

	return Code(coded.ErrorCode()).Domain(), true
}

// InDomain reports whether err carries an error code in the given domain.
// Use it to route errors by failing component without string parsing:
//
//	if httputil.InDomain(err, httputil.Domain("cors")) { ... }
func InDomain(err error, domain Domain) bool {
	found, ok := DomainOf(err)

	return ok && found == domain
}
