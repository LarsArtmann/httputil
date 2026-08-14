package httputil

import (
	"io"
)

func (w *compressWriter) startCompression() error {
	w.Header().Set(headerContentEncoding, w.encoding)
	w.Header().Del(headerContentLength)

	// Pull a writer from this middleware instance's pool (owned by the
	// negotiator and keyed by encoding name). The pool's New function creates
	// a writer bound to io.Discard; we Reset() it to our real writer below to
	// recycle the expensive internal state.
	pool := w.pool
	raw := pool.Get()

	writer, ok := raw.(io.WriteCloser)
	if !ok {
		return codeCompressWriteFailed.WrapTransient(
			errUnexpectedPoolType.WithContextf("pool_element_type", "%T", raw),
			"pool returned unexpected type",
		).WithContext("encoding", w.encoding)
	}

	if resettable, ok := writer.(resettableWriter); ok {
		resettable.Reset(w.ResponseWriter)
	} else {
		// Custom factory without Reset support: fall back to fresh writer.
		fresh, err := w.factory(w.ResponseWriter)
		if err != nil {
			return codeCompressWriteFailed.WrapTransient(
				err,
				"failed to create compression writer",
			).WithContext("encoding", w.encoding)
		}

		writer = fresh
	}

	flusher, ok := writer.(writeCloseFlusher)
	if !ok {
		flusher = nopFlushCloser{writer}
	}

	w.writer = flusher
	w.compressing = true

	w.writeHeaderToUnderlying()

	if len(w.buf) > 0 {
		_, err := w.writer.Write(w.buf)
		w.buf = w.buf[:0]

		if err != nil {
			return codeCompressWriteFailed.WrapTransient(
				err,
				"compression writer buffered write failed",
			).WithContext("encoding", w.encoding)
		}
	}

	return nil
}
