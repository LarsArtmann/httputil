package httputil

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// newPoisonedWriterPool returns a writerPool whose probe reports resettable
// but whose pool yields bare nonWriter values, forcing the wrapper
// type-assertion failure path in acquire deterministically (no reliance on
// sync.Pool retention).
func newPoisonedWriterPool() *writerPool {
	return &writerPool{
		pool:       &sync.Pool{New: func() any { return &nonWriter{} }},
		resettable: true,
	}
}

// newMarkerWriterPool returns a resettable writerPool whose pool yields
// pooledWriter wrappers around fresh markerPoolWriter values owned by the
// pool, making the pooled acquire path deterministic in tests (no reliance on
// sync.Pool retention). factory is only reachable if the pooled path is
// broken; tests pass a must-not-call factory to pin that invariant.
func newMarkerWriterPool(factory WriterFactory) *writerPool {
	var pool *writerPool

	pool = &writerPool{
		factory: factory,
		pool: &sync.Pool{
			New: func() any {
				return &pooledWriter{owner: pool, writer: &markerPoolWriter{}}
			},
		},
		resettable: true,
	}

	return pool
}

// newDirectWriterPool returns a non-resettable writerPool bound to factory,
// for exercising acquire's direct path without newWriterPool's construction
// probe (which panics on factories that fail or return nil).
func newDirectWriterPool(factory WriterFactory) *writerPool {
	return &writerPool{
		factory:    factory,
		pool:       &sync.Pool{},
		resettable: false,
	}
}

// errFactoryMustNotBeCalled signals an acquire path that must never reach
// the pool's factory.
var errFactoryMustNotBeCalled = errors.New("factory must not be called")

// plainWriteCloser is a minimal io.WriteCloser without Reset support,
// exercising the non-resettable factory path of writerPool.
type plainWriteCloser struct {
	io.Writer
}

func (w *plainWriteCloser) Close() error { return nil }

// countingPlainFactory returns a factory producing plainWriteCloser values
// and recording every invocation in calls.
func countingPlainFactory(calls *int) WriterFactory {
	return func(dst io.Writer) (io.WriteCloser, error) {
		*calls++

		return &plainWriteCloser{Writer: dst}, nil
	}
}

// TestWriterPool_NonResettableFactory_SkipsPool verifies that a factory whose
// writers lack Reset(io.Writer) never touches the sync.Pool: every acquire
// builds a fresh writer directly, so custom non-resettable factories pay no
// wasted pooled allocation per request.
func TestWriterPool_NonResettableFactory_SkipsPool(t *testing.T) {
	t.Parallel()

	var calls int

	pool := newWriterPool(countingPlainFactory(&calls)) // construction probe: one call

	first, err := pool.acquire(io.Discard)
	if err != nil {
		t.Fatalf("first acquire() error = %v, want nil", err)
	}

	second, err := pool.acquire(io.Discard)
	if err != nil {
		t.Fatalf("second acquire() error = %v, want nil", err)
	}

	if calls != 3 {
		t.Errorf("factory call count = %d, want 3 (probe + two fresh acquires)", calls)
	}

	if first == second {
		t.Error("acquire returned the same writer instance twice, want distinct instances")
	}
}

// markerPoolWriter is a resettable io.WriteCloser returned by a fake pool
// New, making the resettable (pooled) acquire path deterministic in tests.
type markerPoolWriter struct {
	io.Writer
}

func (w *markerPoolWriter) Close() error { return nil }

func (w *markerPoolWriter) Reset(dst io.Writer) { w.Writer = dst }

// TestWriterPool_ResettableFactory_AcquireUsesPool verifies that a resettable
// writerPool resolves acquire through the sync.Pool (its New) rather than the
// pool's own factory, and that the returned writer is the pool-owned wrapper.
func TestWriterPool_ResettableFactory_AcquireUsesPool(t *testing.T) {
	t.Parallel()

	pool := newMarkerWriterPool(func(io.Writer) (io.WriteCloser, error) {
		t.Error("factory called on the resettable (pooled) acquire path")

		return nil, errFactoryMustNotBeCalled
	})

	writer, err := pool.acquire(io.Discard)
	if err != nil {
		t.Fatalf("acquire() error = %v, want nil", err)
	}

	wrapper, ok := writer.(*pooledWriter)
	if !ok {
		t.Fatalf("acquire returned %T, want a *pooledWriter from the pool", writer)
	}

	if _, ok := wrapper.writer.(*markerPoolWriter); !ok {
		t.Errorf("pool writer = %T, want *markerPoolWriter from the pool", wrapper.writer)
	}

	// A pooled wrapper must always satisfy writeCloseFlusher (Flush by
	// delegation); a wrapped writer without flush support gets the no-op
	// branch, mirroring nopFlushCloser.
	flusher, ok := writer.(writeCloseFlusher)
	if !ok {
		t.Fatal("pooled writer does not satisfy writeCloseFlusher, want delegation Flush")
	}

	if err := flusher.Flush(); err != nil {
		t.Errorf("Flush() on a pooled writer without flush support error = %v, want nil", err)
	}
}

// TestWriterPool_Release_NonResettable_IsNotStored verifies that releasing a
// writer without Reset support never stores it in the pool: the next acquire
// must produce a pool-native writer instead of the dropped instance.
func TestWriterPool_Release_NonResettable_IsNotStored(t *testing.T) {
	t.Parallel()

	pool := newMarkerWriterPool(nil)

	released := &plainWriteCloser{Writer: io.Discard}
	pool.release(released)

	writer, err := pool.acquire(io.Discard)
	if err != nil {
		t.Fatalf("acquire() error = %v, want nil", err)
	}

	if _, ok := writer.(*plainWriteCloser); ok {
		t.Error("acquire returned the released non-resettable writer, want a pool writer")
	}
}

// TestWriterPool_Release_ForeignWrapper_IsNotStored verifies the release
// provenance check: a pooledWriter owned by a different pool is dropped
// instead of stored, so it cannot poison this pool's writer population.
func TestWriterPool_Release_ForeignWrapper_IsNotStored(t *testing.T) {
	t.Parallel()

	poolA := newMarkerWriterPool(nil)
	poolB := newMarkerWriterPool(nil)

	foreign, err := poolA.acquire(io.Discard)
	if err != nil {
		t.Fatalf("poolA acquire() error = %v, want nil", err)
	}

	poolB.release(foreign)

	writer, err := poolB.acquire(io.Discard)
	if err != nil {
		t.Fatalf("poolB acquire() error = %v, want nil", err)
	}

	wrapper, ok := writer.(*pooledWriter)
	if !ok {
		t.Fatalf("poolB acquire returned %T, want a *pooledWriter", writer)
	}

	if wrapper == foreign {
		t.Error("poolB acquire returned the foreign wrapper dropped into it, want a poolB-native writer")
	}

	if wrapper.owner != poolB {
		t.Errorf("poolB acquire returned a wrapper owned by %p, want poolB (%p)", wrapper.owner, poolB)
	}
}

// TestWriterPool_Acquire_ResetsWriterToDestination proves the recycled writer
// is actually Reset to the caller's destination: the probe writer is bound to
// io.Discard, so without the Reset the destination buffer would stay empty.
func TestWriterPool_Acquire_ResetsWriterToDestination(t *testing.T) {
	t.Parallel()

	pool := newWriterPool(GzipWriterFactory(gzip.DefaultCompression))

	var buf bytes.Buffer

	writer, err := pool.acquire(&buf)
	if err != nil {
		t.Fatalf("acquire() error = %v, want nil", err)
	}

	payload := []byte("pooled writer must be reset to the caller destination")

	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	zr, err := gzip.NewReader(&buf)
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

// TestWriterPool_ProbeDetectsResettableWriters pins the construction-time
// probe decision: stdlib gzip writers are resettable (poolable), the
// identity passthrough writer is not.
func TestWriterPool_ProbeDetectsResettableWriters(t *testing.T) {
	t.Parallel()

	gzipPool := newWriterPool(GzipWriterFactory(gzip.DefaultCompression))
	if !gzipPool.resettable {
		t.Error("gzip writer pool resettable = false, want true")
	}

	passthroughPool := newWriterPool(passthroughFactory)
	if passthroughPool.resettable {
		t.Error("passthrough writer pool resettable = true, want false")
	}
}

// TestWriterPool_ProbeNilWriter_PanicsAtConstruction covers the (nil, nil)
// probe branch in newWriterPool: a factory that returns neither a writer nor
// an error is an unrecoverable factory-contract violation, so construction
// panics instead of silently building a pool that fails every request.
func TestWriterPool_ProbeNilWriter_PanicsAtConstruction(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("newWriterPool did not panic on a (nil, nil) factory return")
		}

		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil writer") {
			t.Errorf("panic value = %v, want a factory nil-writer message", r)
		}
	}()

	//nolint:nilnil // deliberately returns (nil, nil): the factory-contract violation under test
	_ = newWriterPool(func(io.Writer) (io.WriteCloser, error) { return nil, nil })
}

// errRefillFactoryFailed signals a refill-time factory error in tests of the
// pool refill constructor's panic branch.
var errRefillFactoryFailed = errors.New("refill factory failed")

// TestWriterPool_RefillFactoryError_PanicsOnAcquire covers the factory-error
// branch of the pool refill constructor: a factory that passes the probe but
// later fails panics inside the wrapped-Get path instead of handing a broken
// writer to a request.
func TestWriterPool_RefillFactoryError_PanicsOnAcquire(t *testing.T) {
	t.Parallel()

	var calls int

	factory := func(io.Writer) (io.WriteCloser, error) {
		calls++
		if calls == 1 {
			return &markerPoolWriter{}, nil
		}

		return nil, errRefillFactoryFailed
	}

	pool := newWriterPool(factory)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("acquire did not panic on a factory error from the pool refill")
		}

		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "writer factory failed") {
			t.Errorf("panic value = %v, want a factory failure message", r)
		}
	}()

	_, _ = pool.acquire(io.Discard)
}

// TestWriterPool_RefillNilWriter_PanicsOnAcquire covers the nil-writer branch
// of the pool refill constructor: a factory that passes the probe (resettable
// writer) but later returns (nil, nil) panics inside the wrapped-Get path
// instead of handing a nil writer to a request.
func TestWriterPool_RefillNilWriter_PanicsOnAcquire(t *testing.T) {
	t.Parallel()

	var calls int

	factory := func(io.Writer) (io.WriteCloser, error) {
		calls++
		if calls == 1 {
			return &markerPoolWriter{}, nil
		}

		//nolint:nilnil // deliberately returns (nil, nil): the factory-contract violation under test
		return nil, nil
	}

	pool := newWriterPool(factory)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("acquire did not panic on a nil writer from the pool refill")
		}

		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil writer") {
			t.Errorf("panic value = %v, want a factory nil-writer message", r)
		}
	}()

	_, _ = pool.acquire(io.Discard)
}

// TestWriterPool_Acquire_ForeignElement_ReturnsClassifiedError covers the
// owner assertion in acquire: an element whose wrapper names a different
// pool surfaces the classified pool-contract error instead of being handed
// to a request.
func TestWriterPool_Acquire_ForeignElement_ReturnsClassifiedError(t *testing.T) {
	t.Parallel()

	foreignOwner := newMarkerWriterPool(nil)

	pool := &writerPool{
		pool: &sync.Pool{
			New: func() any {
				return &pooledWriter{owner: foreignOwner, writer: &markerPoolWriter{}}
			},
		},
		resettable: true,
	}

	writer, err := pool.acquire(io.Discard)
	if writer != nil {
		t.Errorf("acquire writer = %v, want nil", writer)
	}

	if !errors.Is(err, errUnexpectedPoolType) {
		t.Errorf("errors.Is(err, errUnexpectedPoolType) = false, want true")
	}
}

// TestWriterPool_DirectPathNilWriter_ReturnsClassifiedError covers the
// nil-writer guard on acquire's direct (non-pooled) path: a factory that
// passes the probe (non-resettable writer) but later returns (nil, nil)
// surfaces the classified pool-contract error instead of a nil writer.
func TestWriterPool_DirectPathNilWriter_ReturnsClassifiedError(t *testing.T) {
	t.Parallel()

	var calls int

	factory := func(io.Writer) (io.WriteCloser, error) {
		calls++
		if calls == 1 {
			return &plainWriteCloser{Writer: io.Discard}, nil
		}

		//nolint:nilnil // deliberately returns (nil, nil): the factory-contract violation under test
		return nil, nil
	}

	pool := newWriterPool(factory)

	writer, err := pool.acquire(io.Discard)
	if writer != nil {
		t.Errorf("acquire writer = %v, want nil", writer)
	}

	if !errors.Is(err, errUnexpectedPoolType) {
		t.Errorf("errors.Is(err, errUnexpectedPoolType) = false, want true")
	}

	assertClassified(t, err, errorfamily.Infrastructure, false)
}

// TestWriterPool_Acquire_NonResettablePooledElement_ReturnsClassifiedError
// covers the inner-resettable assertion in acquire: a pooled element wrapping
// a writer that lost resettable support surfaces the classified pool-contract
// error instead of a nil-interface Reset panic.
func TestWriterPool_Acquire_NonResettablePooledElement_ReturnsClassifiedError(t *testing.T) {
	t.Parallel()

	var pool *writerPool

	pool = &writerPool{
		pool: &sync.Pool{
			New: func() any {
				return &pooledWriter{owner: pool, writer: &plainWriteCloser{Writer: io.Discard}}
			},
		},
		resettable: true,
	}

	writer, err := pool.acquire(io.Discard)
	if writer != nil {
		t.Errorf("acquire writer = %v, want nil", writer)
	}

	if !errors.Is(err, errUnexpectedPoolType) {
		t.Errorf("errors.Is(err, errUnexpectedPoolType) = false, want true")
	}
}
