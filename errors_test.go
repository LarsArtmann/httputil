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

	family := errorfamily.Classify(err)
	if family != errorfamily.Transient {
		t.Errorf("Classify(err) = %v, want Transient", family)
	}

	if !errorfamily.IsRetryable(err) {
		t.Error("IsRetryable(err) = false, want true")
	}
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

	contextual, ok := errors.AsType[errorfamily.Contextual](err)
	if !ok {
		t.Fatal("error does not implement Contextual")
	}

	ctx := contextual.ErrorContext()
	if ctx["status"] != "200" {
		t.Errorf("context[\"status\"] = %q, want %q", ctx["status"], "200")
	}
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

	family := errorfamily.Classify(err)
	if family != errorfamily.Infrastructure {
		t.Errorf("Classify(err) = %v, want Infrastructure", family)
	}

	if errorfamily.IsRetryable(err) {
		t.Error("IsRetryable(err) = true, want false for Infrastructure")
	}
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

	if !errors.Is(err, http.ErrNotSupported) {
		t.Error("errors.Is(err, http.ErrNotSupported) = false, want true")
	}
}

func TestPush_ReturnsNilError_OnSuccess(t *testing.T) {
	t.Parallel()

	inner := &mockPusher{pushErr: nil}
	recorder := NewResponseRecorder(inner)

	err := recorder.Push("/style.css", nil)
	if err != nil {
		t.Errorf("Push() error = %v, want nil", err)
	}
}

func TestPush_ClassifiedAsInfrastructure_WhenUnsupported(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	err := recorder.Push("/style.css", nil)
	if err == nil {
		t.Fatal("Push() error = nil, want non-nil")
	}

	family := errorfamily.Classify(err)
	if family != errorfamily.Infrastructure {
		t.Errorf("Classify(err) = %v, want Infrastructure", family)
	}

	classified, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		t.Fatal("error does not implement Coded")
	}

	if classified.ErrorCode() != ErrCodePushUnsupported {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), ErrCodePushUnsupported)
	}
}

func TestPush_ClassifiedAsTransient_OnPushFailure(t *testing.T) {
	t.Parallel()

	inner := &mockPusher{pushErr: errPushFailed}
	recorder := NewResponseRecorder(inner)

	err := recorder.Push("/style.css", nil)
	if err == nil {
		t.Fatal("Push() error = nil, want non-nil")
	}

	family := errorfamily.Classify(err)
	if family != errorfamily.Transient {
		t.Errorf("Classify(err) = %v, want Transient", family)
	}

	classified, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		t.Fatal("error does not implement Coded")
	}

	if classified.ErrorCode() != ErrCodePushFailed {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), ErrCodePushFailed)
	}
}

func TestPush_HasTargetContext(t *testing.T) {
	t.Parallel()

	inner := &mockPusher{pushErr: errPushFailed}
	recorder := NewResponseRecorder(inner)

	err := recorder.Push("/style.css", nil)

	contextual, ok := errors.AsType[errorfamily.Contextual](err)
	if !ok {
		t.Fatal("error does not implement Contextual")
	}

	ctx := contextual.ErrorContext()
	if ctx["target"] != "/style.css" {
		t.Errorf("context[\"target\"] = %q, want %q", ctx["target"], "/style.css")
	}
}

func TestPush_ErrNotSupported_InErrorChain_WhenUnsupported(t *testing.T) {
	t.Parallel()

	inner := httptest.NewRecorder()
	recorder := NewResponseRecorder(inner)

	err := recorder.Push("/style.css", nil)

	if !errors.Is(err, http.ErrNotSupported) {
		t.Error("errors.Is(err, http.ErrNotSupported) = false, want true")
	}
}

var (
	errWriteFailed = errors.New("write failed")
	errPushFailed  = errors.New("push failed")
)

type failingWriter struct {
	http.ResponseWriter
}

func (f *failingWriter) Write(b []byte) (int, error) {
	return 0, errWriteFailed
}

type mockPusher struct {
	http.ResponseWriter

	pushErr error
}

func (m *mockPusher) Push(target string, opts *http.PushOptions) error {
	return m.pushErr
}
