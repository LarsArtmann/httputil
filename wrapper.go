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

// pushDelegate forwards an HTTP/2 server push to w when it implements
// http.Pusher. A writer without Push gets a classified Infrastructure error
// (code http.push_unsupported, cause http.ErrNotSupported) so a wrapped
// handler can detect the capability gap instead of silently losing pushes;
// a failing Push surfaces as a Transient, retryable error with the cause
// preserved.
func pushDelegate(w http.ResponseWriter, target string, opts *http.PushOptions) error {
	pusher, ok := w.(http.Pusher)
	if !ok {
		return codePushUnsupported.WrapInfrastructure(
			http.ErrNotSupported,
			"response writer does not implement http.Pusher",
		)
	}

	if err := pusher.Push(target, opts); err != nil {
		return codePushFailed.WrapTransient(
			err,
			"response writer push failed",
		)
	}

	return nil
}

func (w *responseWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hijackDelegate(w.ResponseWriter)
}

func (w *responseWrapper) Flush() {
	flushDelegate(w.ResponseWriter)
}

// Push delegates an HTTP/2 server push to the underlying writer when it
// supports it, keeping the http.Pusher interface alive through the wrapper.
func (w *responseWrapper) Push(target string, opts *http.PushOptions) error {
	return pushDelegate(w.ResponseWriter, target, opts)
}
