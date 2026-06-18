package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestWrite_ReturnsNilError_OnSuccess(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	_, err := recorder.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write() error = %v, want nil", err)
	}
}

func TestWrite_ClassifiedAsTransient_OnFailure(t *testing.T) {
	t.Parallel()

	inner := &failingWriter{}
	recorder := NewResponseRecorder(inner)

	_, err := recorder.Write([]byte("hello"))
	if err == nil {
		t.Fatal("Write() error = nil, want non-nil")
	}

	assertClassified(t, err, errorfamily.Transient, true)
}

func TestWrite_HasErrorCode(t *testing.T) {
	t.Parallel()

	inner := &failingWriter{}
	recorder := NewResponseRecorder(inner)

	_, err := recorder.Write([]byte("hello"))

	classified, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		t.Fatal("error does not implement Coded")
	}

	if classified.ErrorCode() != ErrCodeWriteFailed {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), ErrCodeWriteFailed)
	}
}

func TestWrite_HasStatusContext(t *testing.T) {
	t.Parallel()

	inner := &failingWriter{}
	recorder := NewResponseRecorder(inner)

	_, err := recorder.Write([]byte("hello"))

	assertErrorContext(t, err, "status", "200")
}

func TestWrite_WrapsUnderlyingError(t *testing.T) {
	t.Parallel()

	inner := &failingWriter{}
	recorder := NewResponseRecorder(inner)

	_, err := recorder.Write([]byte("hello"))

	if !errors.Is(err, errWriteFailed) {
		t.Error("errors.Is(err, errWriteFailed) = false, want true")
	}
}

func TestWrite_VerboseFormat(t *testing.T) {
	t.Parallel()

	inner := &failingWriter{}
	recorder := NewResponseRecorder(inner)

	_, err := recorder.Write([]byte("hello"))

	verbose := fmt.Sprintf("%+v", err)
	if verbose == "" {
		t.Error("verbose format is empty")
	}
}

func TestHijack_ReturnsNilError_OnSuccess(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	_, _, err := recorder.Hijack()
	if err == nil {
		t.Error("Hijack() error = nil, but httptest.Recorder doesn't support Hijack")
	}
}

func TestHijack_ClassifiedAsInfrastructure_WhenUnsupported(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	_, _, err := recorder.Hijack()
	if err == nil {
		t.Fatal("Hijack() error = nil, want non-nil")
	}

	assertClassified(t, err, errorfamily.Infrastructure, false)
}

func TestHijack_HasErrorCode_WhenUnsupported(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	_, _, err := recorder.Hijack()

	classified, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		t.Fatal("error does not implement Coded")
	}

	if classified.ErrorCode() != ErrCodeHijackUnsupported {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), ErrCodeHijackUnsupported)
	}
}

func TestHijack_ErrNotSupported_InErrorChain(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	_, _, err := recorder.Hijack()

	assertErrNotSupported(t, err)
}

// TestHijack_Failure_ClassifiedAsTransient covers the path where the
// underlying writer implements Hijacker but its Hijack() returns an error.
func TestHijack_Failure_ClassifiedAsTransient(t *testing.T) {
	t.Parallel()

	recorder := NewResponseRecorder(newFailingHijacker())

	conn, rw, err := recorder.Hijack()
	if conn != nil || rw != nil {
		t.Error("expected nil conn and rw on hijack failure")
	}

	if err == nil {
		t.Fatal("Hijack() error = nil, want non-nil")
	}

	if !errors.Is(err, errHijackFailed) {
		t.Errorf("errors.Is(err, errHijackFailed) = false, want true")
	}

	assertClassified(t, err, errorfamily.Transient, true)

	coded, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		t.Fatal("error does not implement Coded")
	}

	if coded.ErrorCode() != ErrCodeHijackFailed {
		t.Errorf("ErrorCode() = %q, want %q", coded.ErrorCode(), ErrCodeHijackFailed)
	}
}

func assertErrorContext(t *testing.T, err error, key, want string) {
	t.Helper()

	contextual, ok := errors.AsType[errorfamily.Contextual](err)
	if !ok {
		t.Fatal("error does not implement Contextual")
	}

	ctx := contextual.ErrorContext()
	if ctx[key] != want {
		t.Errorf("context[%q] = %q, want %q", key, ctx[key], want)
	}
}

func assertErrNotSupported(t *testing.T, err error) {
	t.Helper()

	if !errors.Is(err, http.ErrNotSupported) {
		t.Error("errors.Is(err, http.ErrNotSupported) = false, want true")
	}
}

var errWriteFailed = errors.New("write failed")

type failingWriter struct {
	http.ResponseWriter
}

func (f *failingWriter) Write(b []byte) (int, error) {
	return 0, errWriteFailed
}

func TestRegisterErrorClassifications_RegistersTemplates(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	family := errorfamily.Classify(http.ErrNotSupported)
	if family != errorfamily.Infrastructure {
		t.Errorf("Classify(ErrNotSupported) = %v, want Infrastructure", family)
	}
}
