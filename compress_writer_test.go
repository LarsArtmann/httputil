package httputil

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
		nil,
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
func assertCompressClassified(t *testing.T, err, sentinel error) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want non-nil")
	}

	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is(err, sentinel) = false, want true")
	}

	assertClassified(t, err, errorfamily.Transient, true)

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

// flakyWriteCloser succeeds on its first Write then fails on every subsequent
// Write. This exercises the streaming-write error path in streamClassified that
// a uniformly-failing writer cannot reach: startCompression's buffered Write
// must succeed so the later streaming Write is the one that fails.
type flakyWriteCloser struct {
	writes int
}

func (w *flakyWriteCloser) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > 1 {
		return 0, errMockCompressWriteFailed
	}

	return len(p), nil
}

func (*flakyWriteCloser) Close() error { return nil }
func (*flakyWriteCloser) Flush() error { return nil }

// failingResponseRecorder is an httptest.ResponseRecorder whose Write always
// fails, exercising the buffered-write error branches in Close and
// flushPlainAndStream that require the underlying ResponseWriter to reject data.
type failingResponseRecorder struct {
	*httptest.ResponseRecorder
}

func (*failingResponseRecorder) Write([]byte) (int, error) {
	return 0, errMockCompressWriteFailed
}

// erroringFactory is a WriterFactory that always returns an error, exercising
// the fresh-writer creation error path in startCompression. It must only be
// used as the compressWriter.factory field, never as a pool source (the pool's
// New panics on factory errors).
var erroringFactory = WriterFactory(func(io.Writer) (io.WriteCloser, error) {
	return nil, errMockCompressWriteFailed
})

// TestCompressWriter_PassthroughWriterRoundTrip drives startCompression through
// the testPassthroughFactory (which yields nopCloserWriter), then exercises
// Flush and Close on that writer. This covers nopCloserWriter.Flush,
// nopCloserWriter.Close, and passthroughFactory — code reachable only via
// direct compressWriter construction since the Compression middleware
// short-circuits the identity encoding.
func TestCompressWriter_PassthroughWriterRoundTrip(t *testing.T) {
	t.Parallel()

	cw := newTestCompressWriter()
	cw.WriteHeader(http.StatusOK)

	err := cw.startCompression()
	if err != nil {
		t.Fatalf("startCompression error = %v", err)
	}

	cw.Flush()

	err = cw.Close()
	if err != nil {
		t.Fatalf("Close error = %v, want nil", err)
	}
}

// TestCompressWriter_StreamWriteError verifies that a streaming Write failure
// (after the buffered Write in startCompression succeeds) surfaces a classified
// error through streamClassified.
func TestCompressWriter_StreamWriteError(t *testing.T) {
	t.Parallel()

	factory := func(io.Writer) (io.WriteCloser, error) { return &flakyWriteCloser{}, nil }
	cw := newCompressWriter(
		newRecorder(),
		1,
		encodingGzip,
		factory,
		&sync.Pool{New: func() any { return &flakyWriteCloser{} }},
		nil,
	)
	cw.WriteHeader(http.StatusOK)

	_, err := cw.Write([]byte("ab"))

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_FlushPlainBufferedWriteError verifies that a buffered
// plain-mode Write failure (non-compressible status) surfaces a classified
// error through flushPlainAndStream.
func TestCompressWriter_FlushPlainBufferedWriteError(t *testing.T) {
	t.Parallel()

	cw := newCompressWriter(
		&failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()},
		defaultCompressionMinSize,
		encodingGzip,
		testPassthroughFactory,
		newWriterPool(testPassthroughFactory),
		nil,
	)
	cw.WriteHeader(http.StatusNotFound)

	body := make([]byte, defaultCompressionMinSize+1)
	_, err := cw.Write(body)

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_CloseBufferedWriteError verifies that a buffered Write
// failure during Close (small response, not compressed, not plain) surfaces a
// classified error.
func TestCompressWriter_CloseBufferedWriteError(t *testing.T) {
	t.Parallel()

	cw := newCompressWriter(
		&failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()},
		defaultCompressionMinSize,
		encodingGzip,
		testPassthroughFactory,
		newWriterPool(testPassthroughFactory),
		nil,
	)
	cw.WriteHeader(http.StatusOK)

	_, _ = cw.Write([]byte("tiny"))

	err := cw.Close()

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_StartCompression_FreshFactoryError verifies that a
// fresh-writer factory error (reached when the pooled writer is not resettable)
// surfaces a classified error.
func TestCompressWriter_StartCompression_FreshFactoryError(t *testing.T) {
	t.Parallel()

	cw := newCompressWriter(
		newRecorder(),
		defaultCompressionMinSize,
		encodingGzip,
		erroringFactory,
		newWriterPool(testPassthroughFactory),
		nil,
	)
	cw.WriteHeader(http.StatusOK)

	err := cw.startCompression()

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_StartCompression_BufferedWriteError verifies that a
// failure while writing the buffered prefix during startCompression surfaces a
// classified error. This also covers the startCompression-error branch of
// startCompressAndStream.
func TestCompressWriter_StartCompression_BufferedWriteError(t *testing.T) {
	t.Parallel()

	factory := func(io.Writer) (io.WriteCloser, error) {
		return &failingCompressWriter{failWrite: true}, nil
	}

	cw := newCompressWriter(
		newRecorder(),
		1,
		encodingGzip,
		factory,
		&sync.Pool{New: func() any { return &failingCompressWriter{} }},
		nil,
	)
	cw.WriteHeader(http.StatusOK)

	_, err := cw.Write([]byte("a"))

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_StartCompressAndStream_PoolError covers the
// startCompression-error propagation branch of startCompressAndStream.
func TestCompressWriter_StartCompressAndStream_PoolError(t *testing.T) {
	t.Parallel()

	cw := newTestCompressWriter()
	cw.pool = &sync.Pool{New: func() any { return &nonWriter{} }}
	cw.WriteHeader(http.StatusOK)

	_, err := cw.startCompressAndStream([]byte("x"), 1)

	assertCompressClassified(t, err, errUnexpectedPoolType)
}

// TestCompressWriter_StartCompressAndStream_EmptyRemainder covers the
// len(b)==0 early-return branch of startCompressAndStream.
func TestCompressWriter_StartCompressAndStream_EmptyRemainder(t *testing.T) {
	t.Parallel()

	cw := newTestCompressWriter()
	cw.WriteHeader(http.StatusOK)

	n, err := cw.startCompressAndStream([]byte{}, 0)
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}

	if n != 0 {
		t.Errorf("returned %d, want 0", n)
	}
}

// TestCompressWriter_FlushInPlainMode covers the plain-mode branch of Flush:
// after the first Flush transitions the writer to plain mode and drains the
// buffer, a second Flush must take the w.plain early-return path.
func TestCompressWriter_FlushInPlainMode(t *testing.T) {
	t.Parallel()

	cw := newTestCompressWriter()
	cw.WriteHeader(http.StatusOK)

	_, _ = cw.Write([]byte("small"))
	cw.Flush()
	cw.Flush()
}

// TestCompressWriter_FlushPlainAndStream_EmptyRemainder covers the len(b)==0
// early-return branch of flushPlainAndStream. This branch is unreachable
// through the public Write path (writeBuffered always forwards a non-empty
// remainder), so it is exercised via a direct method call.
func TestCompressWriter_FlushPlainAndStream_EmptyRemainder(t *testing.T) {
	t.Parallel()

	cw := newTestCompressWriter()
	cw.WriteHeader(http.StatusNotFound)

	_, _ = cw.Write([]byte("x"))

	n, err := cw.flushPlainAndStream([]byte{}, 0)
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}

	if n != 0 {
		t.Errorf("returned %d, want 0", n)
	}
}

// TestNewWriterPool_PanicsOnFactoryError covers the panic branch in
// newWriterPool when the factory fails to construct a writer.
func TestNewWriterPool_PanicsOnFactoryError(t *testing.T) {
	t.Parallel()

	badFactory := WriterFactory(func(io.Writer) (io.WriteCloser, error) {
		return nil, errMockCompressWriteFailed
	})

	pool := newWriterPool(badFactory)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("pool.Get() did not panic on factory error")
		}
	}()

	_ = pool.Get()
}

// TestPassthroughFactory covers passthroughFactory directly. It is part of the
// DefaultWriterFactories contract but never reached through the Compression
// middleware, which short-circuits the identity encoding before the factory is
// invoked.
func TestPassthroughFactory(t *testing.T) {
	t.Parallel()

	w, err := passthroughFactory(io.Discard)
	if err != nil {
		t.Fatalf("passthroughFactory error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Errorf("Close error = %v", err)
	}
}
