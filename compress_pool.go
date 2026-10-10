package httputil

import (
	"io"
	"sync"
)

// errUnexpectedPoolType is returned when a writer pool yields an element that
// violates the pool's writer contract: a non-writer type, an element foreign
// to this pool, a pooled writer that lost its resettable support, or a nil
// writer returned by the factory without an error. Classified as
// Infrastructure: this is a programming error in the WriterFactory contract,
// not a runtime condition.
var errUnexpectedPoolType = codeCompressionPoolTypeUnexpected.Infrastructure(
	"unexpected pool element type",
)

// codeCompressionPoolTypeUnexpected identifies a writer pool whose elements
// do not satisfy the io.WriteCloser contract of the factory that filled it.
const codeCompressionPoolTypeUnexpected = Code("compression.pool_type_unexpected")

// writerPool pairs a per-encoding sync.Pool with a construction-time probe of
// whether the factory's writers are resettable. Factories whose writers
// implement resettableWriter (gzip.Writer, flate.Writer) recycle through the
// pool; every other factory skips the pool entirely so custom non-resettable
// writers do not pay one wasted pooled allocation per request.
//
// The pool owns exactly one factory (bound at construction), and every pooled
// element carries a back-pointer to its owning pool so release can refuse
// writers that did not come from this pool.
type writerPool struct {
	factory    WriterFactory
	pool       *sync.Pool
	resettable bool
}

// pooledWriter is the pool's element wrapper. It records the owning pool so
// release can verify provenance before Put, and it satisfies io.WriteCloser
// (plus a best-effort Flush) by delegation, so acquire can return it directly
// to callers.
type pooledWriter struct {
	owner  *writerPool
	writer io.WriteCloser
}

func (pw *pooledWriter) Write(p []byte) (int, error) {
	return pw.writer.Write(
		p,
	) //nolint:wrapcheck // pure delegation; wrapping happens at the compressWriter choke points
}

func (pw *pooledWriter) Close() error {
	return pw.writer.Close() //nolint:wrapcheck // pure delegation; wrapping happens at the compressWriter choke points
}

// Flush delegates to the wrapped writer when it supports flushing and is a
// no-op otherwise, mirroring the nopFlushCloser fallback for pooled writers.
func (pw *pooledWriter) Flush() error {
	flusher, ok := pw.writer.(writeCloseFlusher)
	if !ok {
		return nil
	}

	return flusher.Flush() //nolint:wrapcheck // pure delegation; wrapping happens at the compressWriter choke points
}

// newWriterPool probes the factory once (bound to io.Discard) to decide
// resettability, then builds the underlying pool whose New constructs fresh
// compression writers the same way. Callers acquire() a writer and use it
// bound to a concrete destination.
//
// The pool is owned by the negotiator for the lifetime of a single Compression
// middleware instance and keyed by encoding name, so it is bounded rather than
// process-global. A global registry keyed by the factory value is impossible
// (function values are not comparable in Go) and keying by the factory's
// parameter address created a fresh entry on every call (a leak).
//
// Panics when the factory fails to construct a writer or returns a nil writer
// without an error: the factory contract is an internal invariant (built-in
// factories never fail), so a failure is an unrecoverable wiring bug, not a
// runtime condition.
func newWriterPool(factory WriterFactory) *writerPool {
	probe, err := factory(io.Discard)
	if err != nil {
		panic("httputil: writer factory failed: " + err.Error())
	}

	if probe == nil {
		panic("httputil: writer factory returned a nil writer without an error")
	}

	_, resettable := probe.(resettableWriter)

	p := &writerPool{
		factory:    factory,
		pool:       &sync.Pool{New: nil}, // bound below, after p exists
		resettable: resettable,
	}
	p.pool.New = p.newPooledWriter

	return p
}

// newPooledWriter is the sync.Pool refill constructor: it builds a fresh
// compression writer via the pool's factory and wraps it with this pool's
// provenance. It panics on factory-contract violations (error or nil writer
// without an error) — the same unrecoverable wiring-bug class as the
// construction probe.
func (p *writerPool) newPooledWriter() any {
	w, err := p.factory(io.Discard)
	if err != nil {
		panic("httputil: writer factory failed: " + err.Error())
	}

	if w == nil {
		panic("httputil: writer factory returned a nil writer without an error")
	}

	return &pooledWriter{owner: p, writer: w}
}

// acquire returns a compression writer bound to dst. Resettable factories
// draw a provenance-checked wrapper from the pool and Reset the recycled
// writer to dst; every other factory builds a fresh writer directly from the
// pool's own factory, because a writer without Reset(io.Writer) cannot be
// rebound and pooling it would only add a wasted allocation per request.
func (p *writerPool) acquire(dst io.Writer) (io.WriteCloser, error) {
	if !p.resettable {
		writer, err := p.factory(dst)
		if err != nil {
			return nil, err
		}

		if writer == nil {
			return nil, errUnexpectedPoolType.WithContextf("pool_element_type", "%T", writer)
		}

		return writer, nil
	}

	raw := p.pool.Get()

	wrapped, ok := raw.(*pooledWriter)
	if !ok {
		return nil, errUnexpectedPoolType.WithContextf("pool_element_type", "%T", raw)
	}

	if wrapped.owner != p {
		return nil, errUnexpectedPoolType.WithContextf("pool_element_type", "%T", raw)
	}

	resettable, ok := wrapped.writer.(resettableWriter)
	if !ok {
		return nil, errUnexpectedPoolType.WithContextf("pool_element_type", "%T", wrapped.writer)
	}

	resettable.Reset(dst)

	return wrapped, nil
}

// release returns a pooled writer to the pool after a successful Close. Only
// writers carrying this pool as their provenance are stored: plain writers
// from non-resettable factories and wrappers from a different pool are
// dropped for garbage collection, so a foreign writer can never poison the
// pool's writer population.
func (p *writerPool) release(writer io.WriteCloser) {
	pw, ok := writer.(*pooledWriter)
	if !ok || pw.owner != p {
		return
	}

	p.pool.Put(pw)
}
