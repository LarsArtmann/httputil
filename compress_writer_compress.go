package httputil

func (w *compressWriter) startCompression() error {
	w.Header().Set(headerContentEncoding, w.encoding)
	w.Header().Del(headerContentLength)

	// Pull a writer from this middleware instance's pool (owned by the
	// negotiator and keyed by encoding name). Resettable factories recycle a
	// pooled writer Reset to our real writer; non-resettable factories build
	// a fresh writer directly, skipping the wasted pooled allocation.
	writer, err := w.pool.acquire(w.ResponseWriter, w.factory)
	if err != nil {
		return codeCompressWriteFailed.WrapTransient(
			err,
			"pool returned unexpected type",
		).WithContext("encoding", w.encoding)
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
