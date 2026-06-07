package httputil

import (
	"bufio"
	"net"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// responseWrapper provides common ResponseWriter wrapping behavior used by
// compressWriter and etagWriter. It buffers WriteHeader calls and delegates
// Hijack, Push, and Flush to the underlying writer when supported.
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

func (w *responseWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errorfamily.WrapInfrastructure(
			http.ErrNotSupported, ErrCodeHijackUnsupported, "response writer does not implement http.Hijacker",
		)
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return conn, rw, errorfamily.WrapTransient(err, ErrCodeHijackFailed, "response writer hijack failed")
	}

	return conn, rw, nil
}

func (w *responseWrapper) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return errorfamily.WrapInfrastructure(
			http.ErrNotSupported, ErrCodePushUnsupported, "response writer does not implement http.Pusher",
		).WithContext("target", target)
	}

	err := pusher.Push(target, opts)
	if err != nil {
		return errorfamily.WrapTransient(err, ErrCodePushFailed, "response writer push failed").
			WithContext("target", target)
	}

	return nil
}

func (w *responseWrapper) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
