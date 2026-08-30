package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// countingHeaderWriter counts WriteHeader calls on the underlying writer so
// tests can assert that buffered statuses are committed exactly once.
type countingHeaderWriter struct {
	*httptest.ResponseRecorder

	writeHeaderCalls int
}

func (c *countingHeaderWriter) WriteHeader(code int) {
	c.writeHeaderCalls++
	c.ResponseRecorder.WriteHeader(code)
}

func newCountingHeaderWriter() *countingHeaderWriter {
	return &countingHeaderWriter{ResponseRecorder: httptest.NewRecorder()}
}

func TestResponseWrapper_WriteHeader_FirstWins(t *testing.T) {
	t.Parallel()

	underlying := newCountingHeaderWriter()
	wrapper := newResponseWrapper(underlying)

	wrapper.WriteHeader(http.StatusCreated)
	wrapper.WriteHeader(http.StatusBadGateway)

	if wrapper.status != http.StatusCreated {
		t.Errorf("wrapper.status = %d, want %d (first WriteHeader wins)", wrapper.status, http.StatusCreated)
	}

	if underlying.writeHeaderCalls != 0 {
		t.Errorf(
			"underlying WriteHeader calls = %d, want 0 (status stays buffered until writeHeaderToUnderlying)",
			underlying.writeHeaderCalls,
		)
	}

	wrapper.writeHeaderToUnderlying()

	if got := underlying.Code; got != http.StatusCreated {
		t.Errorf("underlying status = %d, want %d (first status committed)", got, http.StatusCreated)
	}
}

func TestResponseWrapper_WriteDefaultOK_Commits200WhenNoStatus(t *testing.T) {
	t.Parallel()

	underlying := newRecorder()
	wrapper := newResponseWrapper(underlying)

	wrapper.writeDefaultOK()

	if wrapper.status != http.StatusOK || !wrapper.wroteHeader {
		t.Errorf("after writeDefaultOK: status = %d, wroteHeader = %v, want 200/true",
			wrapper.status, wrapper.wroteHeader)
	}

	wrapper.writeHeaderToUnderlying()

	if got := underlying.Code; got != http.StatusOK {
		t.Errorf("underlying status = %d, want %d (implicit 200 on first Write)", got, http.StatusOK)
	}
}

func TestResponseWrapper_WriteDefaultOK_NoopAfterExplicitStatus(t *testing.T) {
	t.Parallel()

	underlying := newRecorder()
	wrapper := newResponseWrapper(underlying)

	wrapper.WriteHeader(http.StatusNoContent)
	wrapper.writeDefaultOK()

	if wrapper.status != http.StatusNoContent {
		t.Errorf(
			"wrapper.status = %d, want %d (writeDefaultOK must not overwrite)",
			wrapper.status,
			http.StatusNoContent,
		)
	}
}

func TestResponseWrapper_WriteHeaderToUnderlying_CommitsBufferedStatus(t *testing.T) {
	t.Parallel()

	underlying := newRecorder()
	wrapper := newResponseWrapper(underlying)

	wrapper.WriteHeader(http.StatusTeapot)
	wrapper.writeHeaderToUnderlying()

	if got := underlying.Code; got != http.StatusTeapot {
		t.Errorf("underlying status = %d, want %d (buffered status committed)", got, http.StatusTeapot)
	}
}

func TestResponseWrapper_WriteHeaderToUnderlying_NoStatusIsNoop(t *testing.T) {
	t.Parallel()

	underlying := newCountingHeaderWriter()
	wrapper := newResponseWrapper(underlying)

	wrapper.writeHeaderToUnderlying()

	if underlying.writeHeaderCalls != 0 {
		t.Errorf(
			"underlying WriteHeader calls = %d, want 0 (nothing committed before any status)",
			underlying.writeHeaderCalls,
		)
	}
}

func TestResponseWrapper_WriteHeaderToUnderlying_CommitsExactlyOnce(t *testing.T) {
	t.Parallel()

	underlying := newCountingHeaderWriter()
	wrapper := newResponseWrapper(underlying)

	wrapper.WriteHeader(http.StatusAccepted)
	wrapper.writeHeaderToUnderlying()
	wrapper.writeHeaderToUnderlying()

	if underlying.writeHeaderCalls != 1 {
		t.Errorf(
			"underlying WriteHeader calls = %d, want 1 (repeated commit must be a no-op)",
			underlying.writeHeaderCalls,
		)
	}
}

func TestResponseWrapper_Hijack_DelegatesToUnderlying(t *testing.T) {
	t.Parallel()

	underlying := newHijackRecorder()
	wrapper := newResponseWrapper(underlying)

	if _, _, err := wrapper.Hijack(); err != nil {
		t.Fatalf("Hijack() error = %v, want nil", err)
	}

	if !underlying.hijacked {
		t.Error("underlying hijacked = false, want true (Hijack must delegate)")
	}
}

func TestResponseWrapper_Hijack_UnsupportedUnderlying(t *testing.T) {
	t.Parallel()

	wrapper := newResponseWrapper(newRecorder())

	conn, rw, err := wrapper.Hijack()

	if !errors.Is(err, http.ErrNotSupported) {
		t.Errorf("Hijack() error = %v, want ErrNotSupported chain", err)
	}

	if conn != nil || rw != nil {
		t.Error("Hijack() returned non-nil conn/rw for non-hijacker underlying")
	}
}

func TestResponseWrapper_Flush_DelegatesToUnderlying(t *testing.T) {
	t.Parallel()

	underlying := newRecorder()
	wrapper := newResponseWrapper(underlying)

	wrapper.Flush()

	if !underlying.Flushed {
		t.Error("underlying Flushed = false, want true (Flush must delegate)")
	}
}
