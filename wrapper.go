package httputil

import (
	"bufio"
	"net"
	"net/http"
)

// responseWrapper provides common ResponseWriter wrapping behavior used by
// compressWriter. It buffers WriteHeader calls and delegates
// Hijack and Flush to the underlying writer when supported.
type responseWrapper struct {
	http.ResponseWriter

	status        int
	wroteHeader   bool
	headerWritten bool
}

func newResponseWrapper(resp http.ResponseWriter) responseWrapper {
	return responseWrapper{
		ResponseWriter: resp,
		status:         0,
		wroteHeader:    false,
		headerWritten:  false,
	}
}

func (w *responseWrapper) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
}

func (w *responseWrapper) writeHeaderToUnderlying() {
	if w.wroteHeader && !w.headerWritten {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerWritten = true
	}
}

// writeDefaultOK commits a 200 OK status to the underlying ResponseWriter if
// no status has been written yet. Use this at the top of Write methods on
// wrapper types to honor Go's net/http contract: the first Write implicitly
// sends 200 if WriteHeader was not called.
func (w *responseWrapper) writeDefaultOK() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
}

func hijackDelegate(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, codeHijackUnsupported.WrapInfrastructure(
			http.ErrNotSupported,
			"response writer does not implement http.Hijacker",
		)
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return conn, rw, codeHijackFailed.WrapTransient(
			err,
			"response writer hijack failed",
		)
	}

	return conn, rw, nil
}

func flushDelegate(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *responseWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hijackDelegate(w.ResponseWriter)
}

func (w *responseWrapper) Flush() {
	flushDelegate(w.ResponseWriter)
}
