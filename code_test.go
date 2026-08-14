package httputil

import (
	"errors"
	"fmt"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	errTestPlain = errors.New("plain")
	errTestBoom  = errors.New("boom")
)

func TestCodeDomain(t *testing.T) {
	t.Parallel()

	got := Code("cors.max_age_negative").Domain()
	if got != Domain("cors") {
		t.Errorf("Code.Domain() = %q, want %q", got, "cors")
	}
}

func TestCodeDomainWithoutDot(t *testing.T) {
	t.Parallel()

	got := Code("nocode").Domain()
	if got != Domain("nocode") {
		t.Errorf("Code.Domain() = %q, want %q", got, "nocode")
	}
}

func TestCodeDomainEmpty(t *testing.T) {
	t.Parallel()

	got := Code("").Domain()
	if got != Domain("") {
		t.Errorf("Code.Domain() = %q, want empty domain", got)
	}
}

func TestCodeConstructorFamilies(t *testing.T) {
	t.Parallel()

	code := Code("test.constructor")

	rejection := code.Rejection("msg")
	if rejection.ErrorFamily() != errorfamily.Rejection {
		t.Errorf("Rejection() family = %v, want %v", rejection.ErrorFamily(), errorfamily.Rejection)
	}

	transient := code.Transient("msg")
	if transient.ErrorFamily() != errorfamily.Transient {
		t.Errorf("Transient() family = %v, want %v", transient.ErrorFamily(), errorfamily.Transient)
	}

	infrastructure := code.Infrastructure("msg")
	if infrastructure.ErrorFamily() != errorfamily.Infrastructure {
		t.Errorf(
			"Infrastructure() family = %v, want %v",
			infrastructure.ErrorFamily(),
			errorfamily.Infrastructure,
		)
	}
}

func TestCodeConstructorSetsCodeAndMessage(t *testing.T) {
	t.Parallel()

	err := Code("test.detail").Rejection("the message")
	if err.ErrorCode() != "test.detail" {
		t.Errorf("ErrorCode() = %q, want %q", err.ErrorCode(), "test.detail")
	}

	if err.Message() != "the message" {
		t.Errorf("Message() = %q, want %q", err.Message(), "the message")
	}
}

func TestCodeWrapPreservesCause(t *testing.T) {
	t.Parallel()

	cause := errTestBoom
	err := Code("test.wrap").WrapTransient(cause, "wrapper message")

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}

	if err.ErrorFamily() != errorfamily.Transient {
		t.Errorf("family = %v, want %v", err.ErrorFamily(), errorfamily.Transient)
	}
}

func TestDomainOfClassifiedError(t *testing.T) {
	t.Parallel()

	err := Code("server.read_timeout_negative").Rejection("msg")

	domain, ok := DomainOf(err)
	if !ok {
		t.Fatal("DomainOf() ok = false, want true")
	}

	if domain != Domain("server") {
		t.Errorf("DomainOf() = %q, want %q", domain, "server")
	}
}

func TestDomainOfWrappedError(t *testing.T) {
	t.Parallel()

	inner := Code("cors.max_age_negative").Rejection("inner")
	outer := fmt.Errorf("outer: %w", inner)

	domain, ok := DomainOf(outer)
	if !ok {
		t.Fatal("DomainOf(wrapped) ok = false, want true")
	}

	if domain != Domain("cors") {
		t.Errorf("DomainOf(wrapped) = %q, want %q", domain, "cors")
	}
}

func TestDomainOfUncodedError(t *testing.T) {
	t.Parallel()

	domain, ok := DomainOf(errTestPlain)
	if ok {
		t.Errorf("DomainOf(plain) ok = true with domain %q, want false", domain)
	}
}

func TestDomainOfNil(t *testing.T) {
	t.Parallel()

	domain, ok := DomainOf(nil)
	if ok {
		t.Errorf("DomainOf(nil) ok = true with domain %q, want false", domain)
	}
}

func TestInDomainMatch(t *testing.T) {
	t.Parallel()

	err := Code("csrf.samesite_insecure").Infrastructure("msg")

	if !InDomain(err, Domain("csrf")) {
		t.Errorf("InDomain(csrf error, csrf) = false, want true")
	}
}

func TestInDomainMismatch(t *testing.T) {
	t.Parallel()

	err := Code("csrf.samesite_insecure").Infrastructure("msg")

	if InDomain(err, Domain("cors")) {
		t.Errorf("InDomain(csrf error, cors) = true, want false")
	}
}

func TestInDomainUncodedError(t *testing.T) {
	t.Parallel()

	if InDomain(errTestPlain, Domain("cors")) {
		t.Errorf("InDomain(plain, cors) = true, want false")
	}
}
