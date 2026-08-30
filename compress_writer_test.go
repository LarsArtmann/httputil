package httputil

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

var errMockCompressWriteFailed = errors.New("mock compression write failed")

var errMockCompressCloseFailed = errors.New("mock compression close failed")

// newTestCompressWriter builds a compressWriter backed by passthroughFactory
// and a fresh writer pool, for unit tests that manipulate writer state directly.
func newTestCompressWriter() *compressWriter {
	return newCompressWriter(
		newRecorder(),
		defaultCompressionMinSize,
		encodingGzip,
		passthroughFactory,
		newWriterPool(passthroughFactory),
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

// TestCompressWriter_CloseTwice_IsIdempotent verifies that calling Close
// twice on a compressing writer does not panic and reports no error on the
// second call: the first Close nils the writer and clears the compressing
// flag, so the second Close takes the buffered no-op path.
func TestCompressWriter_CloseTwice_IsIdempotent(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.compressing = true
	compressWriter.writer = &failingCompressWriter{}

	firstErr := compressWriter.Close()
	if firstErr != nil {
		t.Fatalf("first Close() error = %v, want nil", firstErr)
	}

	secondErr := compressWriter.Close()
	if secondErr != nil {
		t.Fatalf("second Close() error = %v, want nil", secondErr)
	}
}

// TestCompressWriter_CloseErrorThenCloseAgain_StaysIdempotent verifies that
// after a failing compression-writer Close, the state is still reset so a
// second Close neither panics nor re-reports the failure.
func TestCompressWriter_CloseErrorThenCloseAgain_StaysIdempotent(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.compressing = true
	compressWriter.writer = &failingCompressWriter{failClose: true}

	firstErr := compressWriter.Close()

	assertCompressClassified(t, firstErr, errMockCompressCloseFailed)

	secondErr := compressWriter.Close()
	if secondErr != nil {
		t.Fatalf("second Close() error = %v, want nil", secondErr)
	}
}

// TestCompressWriter_ExactMinSizeWrite_IsNotDuplicated pins the regression
// found by FuzzCompression (2026-08-30): a Write that exactly fills the
// buffer to minSize used to fall through with the payload unconsumed, so the
// bytes were compressed and emitted twice. The decoded stream must equal the
// written payload exactly.
func TestCompressWriter_ExactMinSizeWrite_IsNotDuplicated(t *testing.T) {
	t.Parallel()

	factory := GzipWriterFactory(gzip.DefaultCompression)
	rec := newRecorder()
	rec.Header().Set("Content-Type", "text/plain")

	compressWriter := newCompressWriter(
		rec,
		8,
		encodingGzip,
		factory,
		newWriterPool(factory),
		nil,
	)

	payload := []byte("01234567") // exactly minSize bytes

	if _, err := compressWriter.Write(payload); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if err := compressWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v, want nil", err)
	}

	decoded, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip decode error = %v, want nil", err)
	}

	if !bytes.Equal(decoded, payload) {
		t.Errorf(
			"decoded body = %q (%d bytes), want %q (%d bytes)",
			decoded,
			len(decoded),
			payload,
			len(payload),
		)
	}
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

func (w *flakyWriteCloser) Write(data []byte) (int, error) {
	w.writes++
	if w.writes > 1 {
		return 0, errMockCompressWriteFailed
	}

	return len(data), nil
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
// the passthroughFactory (which yields nopCloserWriter), then exercises
// Flush and Close on that writer. This covers nopCloserWriter.Flush,
// nopCloserWriter.Close, and passthroughFactory — code reachable only via
// direct compressWriter construction since the Compression middleware
// short-circuits the identity encoding.
func TestCompressWriter_PassthroughWriterRoundTrip(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.WriteHeader(http.StatusOK)

	err := compressWriter.startCompression()
	if err != nil {
		t.Fatalf("startCompression error = %v", err)
	}

	compressWriter.Flush()

	err = compressWriter.Close()
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
	compressWriter := newCompressWriter(
		newRecorder(),
		1,
		encodingGzip,
		factory,
		&sync.Pool{New: func() any { return &flakyWriteCloser{} }},
		nil,
	)
	compressWriter.WriteHeader(http.StatusOK)

	_, err := compressWriter.Write([]byte("ab"))

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_FlushPlainBufferedWriteError verifies that a buffered
// plain-mode Write failure (non-compressible status) surfaces a classified
// error through flushPlainAndStream.
func TestCompressWriter_FlushPlainBufferedWriteError(t *testing.T) {
	t.Parallel()

	compressWriter := newCompressWriter(
		&failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()},
		defaultCompressionMinSize,
		encodingGzip,
		passthroughFactory,
		newWriterPool(passthroughFactory),
		nil,
	)
	compressWriter.WriteHeader(http.StatusNotFound)

	body := strings.Repeat("x", defaultCompressionMinSize+1)
	_, err := compressWriter.Write([]byte(body))

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_CloseBufferedWriteError verifies that a buffered Write
// failure during Close (small response, not compressed, not plain) surfaces a
// classified error.
func TestCompressWriter_CloseBufferedWriteError(t *testing.T) {
	t.Parallel()

	compressWriter := newCompressWriter(
		&failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()},
		defaultCompressionMinSize,
		encodingGzip,
		passthroughFactory,
		newWriterPool(passthroughFactory),
		nil,
	)
	compressWriter.WriteHeader(http.StatusOK)

	_, _ = compressWriter.Write([]byte("tiny"))

	err := compressWriter.Close()

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_StartCompression_FreshFactoryError verifies that a
// fresh-writer factory error (reached when the pooled writer is not resettable)
// surfaces a classified error.
func TestCompressWriter_StartCompression_FreshFactoryError(t *testing.T) {
	t.Parallel()

	compressWriter := newCompressWriter(
		newRecorder(),
		defaultCompressionMinSize,
		encodingGzip,
		erroringFactory,
		newWriterPool(passthroughFactory),
		nil,
	)
	compressWriter.WriteHeader(http.StatusOK)

	err := compressWriter.startCompression()

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

	compressWriter := newCompressWriter(
		newRecorder(),
		1,
		encodingGzip,
		factory,
		&sync.Pool{New: func() any { return &failingCompressWriter{} }},
		nil,
	)
	compressWriter.WriteHeader(http.StatusOK)

	_, err := compressWriter.Write([]byte("a"))

	assertCompressClassified(t, err, errMockCompressWriteFailed)
}

// TestCompressWriter_StartCompressAndStream_PoolError covers the
// startCompression-error propagation branch of startCompressAndStream.
func TestCompressWriter_StartCompressAndStream_PoolError(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.pool = &sync.Pool{New: func() any { return &nonWriter{} }}
	compressWriter.WriteHeader(http.StatusOK)

	_, err := compressWriter.startCompressAndStream([]byte("x"), 1)

	assertCompressClassified(t, err, errUnexpectedPoolType)
}

// TestCompressWriter_StartCompressAndStream_EmptyRemainder covers the
// len(b)==0 early-return branch of startCompressAndStream.
func TestCompressWriter_StartCompressAndStream_EmptyRemainder(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.WriteHeader(http.StatusOK)

	n, err := compressWriter.startCompressAndStream([]byte{}, 0)
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

	rec := newRecorder()
	compressWriter := newCompressWriter(
		rec,
		defaultCompressionMinSize,
		encodingGzip,
		passthroughFactory,
		newWriterPool(passthroughFactory),
		nil,
	)
	compressWriter.WriteHeader(http.StatusOK)

	_, _ = compressWriter.Write([]byte("small"))
	compressWriter.Flush()

	if !compressWriter.plain {
		t.Error("writer did not enter plain mode after buffered Flush")
	}

	compressWriter.Flush()

	if got := rec.Body.String(); got != "small" {
		t.Errorf("flushed body = %q, want %q", got, "small")
	}
}

// TestCompressWriter_FlushPlainAndStream_EmptyRemainder covers the len(b)==0
// early-return branch of flushPlainAndStream. This branch is unreachable
// through the public Write path (writeBuffered always forwards a non-empty
// remainder), so it is exercised via a direct method call.
func TestCompressWriter_FlushPlainAndStream_EmptyRemainder(t *testing.T) {
	t.Parallel()

	compressWriter := newTestCompressWriter()
	compressWriter.WriteHeader(http.StatusNotFound)

	_, _ = compressWriter.Write([]byte("x"))

	n, err := compressWriter.flushPlainAndStream([]byte{}, 0)
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}

	if n != 0 {
		t.Errorf("returned %d, want 0", n)
	}
}

// TestNewWriterPool_PanicsOnFactoryError covers the panic branch in
// newWriterPool when the factory fails to construct a writer. Reuses
// erroringFactory rather than a duplicate local factory.
func TestNewWriterPool_PanicsOnFactoryError(t *testing.T) {
	t.Parallel()

	pool := newWriterPool(erroringFactory)

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

	closeErr := w.Close()
	if closeErr != nil {
		t.Errorf("Close error = %v", closeErr)
	}
}
