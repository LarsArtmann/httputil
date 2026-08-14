package httputil

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"sync"
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
	skipTypes   []string
	buf         []byte
	compressing bool
	plain       bool
	writer      writeCloseFlusher
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
	skipTypes []string,
) *compressWriter {
	bufCap := max(minSize, defaultCompressionMinSize)

	return &compressWriter{
		responseWrapper: newResponseWrapper(resp),
		encoding:        encoding,
		factory:         factory,
		pool:            pool,
		minSize:         minSize,
		skipTypes:       skipTypes,
		buf:             make([]byte, 0, bufCap),
		compressing:     false,
		plain:           false,
		writer:          nil,
	}
}

func (w *compressWriter) Write(b []byte) (int, error) {
	w.writeDefaultOK()

	switch {
	case w.plain:
		return w.writeClassified(w.ResponseWriter, b, "failed to write to response writer")
	case w.compressing:
		return w.writeClassified(w.writer, b, "compression writer write failed")
	default:
		return w.writeBuffered(b)
	}
}

// compressWriteError classifies err as a Transient, retryable compress-writer
// failure (ErrCodeCompressWriteFailed) annotated with the negotiated encoding
// for diagnostics. It is the single wrapping site so every Write and Close
// error path reports a consistent family, code, and context.
func (w *compressWriter) compressWriteError(err error, message string) error {
	return codeCompressWriteFailed.WrapTransient(
		err,
		message,
	).WithContext("encoding", w.encoding)
}

// writeClassified is the Write-path error-handling choke point for
// compressWriter: it writes b to dst and, on failure, returns the bytes
// written plus a classified error. Routing the main Write-path fallible
// writes through here keeps the WrapTransient + encoding annotation in one
// place instead of duplicated across the plain and compressed code paths.
// Buffer-drain writes in Close and flushPlainAndStream call
// compressWriteError directly because they need different return semantics.
func (w *compressWriter) writeClassified(dst io.Writer, b []byte, message string) (int, error) {
	n, err := dst.Write(b)
	if err != nil {
		return n, w.compressWriteError(err, message)
	}

	return n, nil
}

// streamClassified writes b to dst but reports total (the full payload size)
// rather than bytes written, so buffered-then-streamed callers can advertise
// complete consumption while still surfacing the classified write error.
func (w *compressWriter) streamClassified(
	dst io.Writer,
	b []byte,
	total int,
	message string,
) (int, error) {
	_, err := w.writeClassified(dst, b, message)
	if err != nil {
		return total, err
	}

	return total, nil
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

	return w.streamClassified(w.writer, b, total, "compression writer streaming write failed")
}

// flushPlainAndStream switches the writer into plain (uncompressed) mode,
// drains any buffered bytes, then streams the remaining payload through
// the underlying ResponseWriter.
func (w *compressWriter) flushPlainAndStream(b []byte, total int) (int, error) {
	w.beginPlainResponse()

	if len(w.buf) > 0 {
		_, err := w.ResponseWriter.Write(w.buf)
		if err != nil {
			return 0, w.compressWriteError(err, "failed to write buffered plain response")
		}

		w.buf = w.buf[:0]
	}

	if len(b) == 0 {
		return total, nil
	}

	return w.streamClassified(w.ResponseWriter, b, total, "failed to write plain response")
}

func (w *compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.beginPlainResponse()

	return w.responseWrapper.Hijack()
}

func (w *compressWriter) shouldCompress() bool {
	if w.status < http.StatusOK || w.status >= http.StatusMultipleChoices {
		return false
	}

	if w.Header().Get(headerContentEncoding) != "" {
		return false
	}

	if !isCompressibleContentType(w.Header().Get(headerContentType), w.skipTypes) {
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
			return w.compressWriteError(err, "compression writer close failed")
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
		_, err := w.ResponseWriter.Write(w.buf)
		if err != nil {
			return w.compressWriteError(err, "buffered response write failed")
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

	w.beginPlainResponse()

	// Post-header-commit body writes are fundamentally unreportable: the
	// handler has returned and the HTTP response is already in-flight.
	// Any write failure here cannot be surfaced to the client or caller.
	if len(w.buf) > 0 {
		_, _ = w.ResponseWriter.Write(w.buf)

		w.buf = w.buf[:0]
	}

	w.responseWrapper.Flush()
}

// beginPlainResponse transitions the writer into plain (uncompressed) mode
// and commits any pending status header to the underlying ResponseWriter.
// Callers must subsequently drain w.buf and forward further writes to the
// underlying ResponseWriter directly.
func (w *compressWriter) beginPlainResponse() {
	w.plain = true
	w.writeHeaderToUnderlying()
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
