package httputil

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

// compressWriter wraps an http.ResponseWriter, buffering small responses to
// avoid the compression cost on tiny payloads, then streaming the remainder
// through a pluggable compression factory.
//
// It delegates to a WriterFactory supplied at construction time. The factory
// pattern supports gzip, deflate, brotli, zstd, or any custom encoding.
type compressWriter struct {
	responseWrapper

	encoding    string
	factory     WriterFactory
	pool        *sync.Pool
	minSize     int
	buf         []byte
	compressing bool
	plain       bool
	writer      writeCloseFlusher // concrete compress writer (gzip, flate, custom)
}

// writeCloseFlusher combines io.WriteCloser and http.Flusher so we can
// surface Flush() on gzip.Writer and flate.Writer.
type writeCloseFlusher interface {
	Write(p []byte) (int, error)
	Close() error
	Flush() error
}

// resettableWriter is an optional interface that compression writers can
// implement to support pooling. gzip.Writer and flate.Writer both implement
// Reset(io.Writer).
type resettableWriter interface {
	Reset(destination io.Writer)
}

func newCompressWriter(
	resp http.ResponseWriter,
	minSize int,
	encoding string,
	factory WriterFactory,
	pool *sync.Pool,
) *compressWriter {
	// Pre-allocate buf to minSize capacity. This avoids 2-3 intermediate
	// reallocations as the slice grows from 0 to minSize via append.
	bufCap := max(minSize, defaultCompressionMinSize)

	return &compressWriter{
		responseWrapper: newResponseWrapper(resp),
		encoding:        encoding,
		factory:         factory,
		pool:            pool,
		minSize:         minSize,
		buf:             make([]byte, 0, bufCap),
		compressing:     false,
		plain:           false,
		writer:          nil,
	}
}

func (w *compressWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	switch {
	case w.plain:
		return w.writePlain(b)
	case w.compressing:
		return w.writeCompressed(b)
	default:
		return w.writeBuffered(b)
	}
}

func (w *compressWriter) writePlain(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if err != nil {
		return n, errorfamily.WrapTransient(
			err,
			ErrCodeCompressWriteFailed,
			"failed to write to response writer",
		).WithContext("encoding", w.encoding)
	}

	return n, nil
}

func (w *compressWriter) writeCompressed(b []byte) (int, error) {
	n, err := w.writer.Write(b)
	if err != nil {
		return n, errorfamily.WrapTransient(
			err,
			ErrCodeCompressWriteFailed,
			"compression writer write failed",
		).WithContext("encoding", w.encoding)
	}

	return n, nil
}

func (w *compressWriter) writeBuffered(b []byte) (int, error) {
	total := len(b)

	// Buffer only up to minSize so large responses do not accumulate
	// indefinitely before the compress-or-not decision is made.
	needed := w.minSize - len(w.buf)
	if needed > 0 {
		if len(b) <= needed {
			w.buf = append(w.buf, b...)

			if len(w.buf) < w.minSize {
				return total, nil
			}
		} else {
			w.buf = append(w.buf, b[:needed]...)
			b = b[needed:]
		}
	}

	if w.shouldCompress() {
		return w.startCompressAndStream(b, total)
	}

	return w.flushPlainAndStream(b, total)
}

func (w *compressWriter) startCompressAndStream(b []byte, total int) (int, error) {
	err := w.startCompression()
	if err != nil {
		return 0, err
	}

	if len(b) == 0 {
		return total, nil
	}

	_, err = w.writer.Write(b)
	if err != nil {
		return total, errorfamily.WrapTransient(
			err,
			ErrCodeCompressWriteFailed,
			"compression writer streaming write failed",
		).WithContext("encoding", w.encoding)
	}

	return total, nil
}

func (w *compressWriter) flushPlainAndStream(b []byte, total int) (int, error) {
	w.plain = true
	w.writeHeaderToUnderlying()

	if len(w.buf) > 0 {
		//nolint:gosec // w.buf is response body content (not user-influenced
		// in an XSS context); G705 taint analysis is a false positive here.
		_, err := w.ResponseWriter.Write(w.buf)
		if err != nil {
			return 0, errorfamily.WrapTransient(
				err,
				ErrCodeCompressWriteFailed,
				"failed to write buffered plain response",
			)
		}

		w.buf = w.buf[:0]
	}

	if len(b) == 0 {
		return total, nil
	}

	_, err := w.ResponseWriter.Write(b)
	if err != nil {
		return total, errorfamily.WrapTransient(
			err,
			ErrCodeCompressWriteFailed,
			"failed to write plain response",
		)
	}

	return total, nil
}

func (w *compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.plain = true

	return w.responseWrapper.Hijack()
}

func (w *compressWriter) shouldCompress() bool {
	if w.status < http.StatusOK || w.status >= http.StatusMultipleChoices {
		return false
	}

	if w.Header().Get(headerContentEncoding) != "" {
		return false
	}

	if !isCompressibleContentType(w.Header().Get(headerContentType)) {
		return false
	}

	return true
}

func (w *compressWriter) Close() error {
	if w.plain {
		return nil
	}

	if w.compressing {
		err := w.writer.Close()
		if err != nil {
			return errorfamily.WrapTransient(
				err,
				ErrCodeCompressWriteFailed,
				"compression writer close failed",
			).WithContext("encoding", w.encoding)
		}

		// Return the writer to the per-factory pool. Only writers that
		// implement resettableWriter (gzip.Writer, flate.Writer) are
		// poolable; others are released for GC.
		if _, resettable := w.writer.(resettableWriter); resettable {
			w.pool.Put(w.writer)
		}

		w.writer = nil

		return nil
	}

	w.writeHeaderToUnderlying()

	if len(w.buf) > 0 {
		//nolint:gosec // w.buf is response body content (not user-influenced
		// in an XSS context); G705 taint analysis is a false positive here.
		_, err := w.ResponseWriter.Write(w.buf)
		if err != nil {
			return errorfamily.WrapTransient(
				err,
				ErrCodeCompressWriteFailed,
				"buffered response write failed",
			)
		}
	}

	return nil
}

func (w *compressWriter) Flush() {
	if w.compressing {
		_ = w.writer.Flush()

		w.responseWrapper.Flush()

		return
	}

	if w.plain {
		w.responseWrapper.Flush()

		return
	}

	w.plain = true

	w.writeHeaderToUnderlying()

	if len(w.buf) > 0 {
		_, _ = w.ResponseWriter.Write(w.buf)
		w.buf = w.buf[:0]
	}

	w.responseWrapper.Flush()
}

// nopCloserWriter wraps an io.Writer and implements io.WriteCloser with a
// no-op Close. Used for the "identity" encoding (passthrough).
type nopCloserWriter struct {
	io.Writer
}

func (nopCloserWriter) Close() error { return nil }
func (nopCloserWriter) Flush() error { return nil }

// nopFlushCloser wraps a WriteCloser and adds a no-op Flush for encodings
// that don't support flushing.
type nopFlushCloser struct {
	io.WriteCloser
}

func (nopFlushCloser) Flush() error { return nil }
