package httputil

import (
	"errors"
	"io"
	"sync"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

var errMockCompressWriteFailed = errors.New("mock compression write failed")

var errMockCompressCloseFailed = errors.New("mock compression close failed")

// testPassthroughFactory is a minimal WriterFactory used by compressWriter
// unit tests where a real encoder is not exercised.
var testPassthroughFactory = WriterFactory(func(dst io.Writer) (io.WriteCloser, error) {
	return nopCloserWriter{dst}, nil
})

// newTestCompressWriter builds a compressWriter backed by testPassthroughFactory
// and a fresh writer pool, for unit tests that manipulate writer state directly.
func newTestCompressWriter() *compressWriter {
	return newCompressWriter(
		newRecorder(),
		defaultCompressionMinSize,
		encodingGzip,
		testPassthroughFactory,
		newWriterPool(testPassthroughFactory),
	)
}

// failingCompressWriter is a writeCloseFlusher double whose Write and Close
// fail on demand, exercising the error branches of compressWriter.
type failingCompressWriter struct {
	failWrite bool
	failClose bool
}

func (f *failingCompressWriter) Write(p []byte) (int, error) {
	if f.failWrite {
		return 0, errMockCompressWriteFailed
	}

	return len(p), nil
}

func (f *failingCompressWriter) Close() error {
	if f.failClose {
		return errMockCompressCloseFailed
	}

	return nil
}

func (f *failingCompressWriter) Flush() error { return nil }

// nonWriter is a non-writeCloseFlusher value used to poison a writer pool.
// It must not implement io.WriteCloser so the pool type assertion fails.
type nonWriter struct{}

// assertCompressClassified verifies err is a Transient, retryable error
// carrying ErrCodeCompressWriteFailed and wrapping the given sentinel.
func assertCompressClassified(t *testing.T, err error, sentinel error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want non-nil")
	}

	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, sentinel) = false, want true")
	}

	if errorfamily.Classify(err) != errorfamily.Transient {
		t.Errorf("Classify(err) = %v, want Transient", errorfamily.Classify(err))
	}

	if !errorfamily.IsRetryable(err) {
		t.Error("IsRetryable(err) = false, want true")
	}

	coded, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		t.Fatal("error does not implement Coded")
	}

	if coded.ErrorCode() != ErrCodeCompressWriteFailed {
		t.Errorf("ErrorCode() = %q, want %q", coded.ErrorCode(), ErrCodeCompressWriteFailed)
	}
}

// TestCompressWriter_WriteCompressed_ReturnsClassifiedError verifies that a
// failing compressed Write surfaces a classified Transient error rather than
// the bare underlying error.
func TestCompressWriter_WriteCompressed_ReturnsClassifiedError(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.compressing = true
	compressWriter.writer = &failingCompressWriter{failWrite: true}

	_, err := compressWriter.Write([]byte("data"))

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_Close_ReturnsClassifiedError verifies that a failing
// compression-writer Close surfaces a classified Transient error.
func TestCompressWriter_Close_ReturnsClassifiedError(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.compressing = true
	compressWriter.writer = &failingCompressWriter{failClose: true}

	err := compressWriter.Close()

	assertCompressClassified(t, err, errMockCompressCloseFailed)
}

// TestCompressWriter_StartCompression_PoolTypeMismatch verifies that a pooled
// object of an unexpected type produces a classified error instead of a panic.
// The pool's New always returns a non-writeCloseFlusher value, so Get() is
// deterministic regardless of GC (no reliance on sync.Pool retention).
func TestCompressWriter_StartCompression_PoolTypeMismatch(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()

	// *nonWriter does not implement io.WriteCloser, so the type assertion in
	// startCompression fails on every Get().
	compressWriter.pool = &sync.Pool{New: func() any { return &nonWriter{} }}

	err := compressWriter.startCompression()

	if !errors.Is(err, errUnexpectedPoolType) {
		t.Errorf("errors.Is(err, errUnexpectedPoolType) = false, want true")
	}

	assertCompressClassified(t, err, errUnexpectedPoolType)
}

// TestNegotiator_PoolsAreStablePerEncoding guards against the regression where
// writer pools were keyed by a parameter address, leaking one pool per request
// and never reusing writers. The same encoding must resolve to the same pool.
func TestNegotiator_PoolsAreStablePerEncoding(t *testing.T) {
	t.Parallel()

	neg := buildNegotiator(DefaultWriterFactories())

	first := neg.poolFor(encodingGzip)
	second := neg.poolFor(encodingGzip)

	if first == nil {
		t.Fatal("poolFor(gzip) = nil, want non-nil")
	}

	if first != second {
		t.Error("poolFor(gzip) returned distinct pools for the same encoding")
	}
}
