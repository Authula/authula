package fiber

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// fiberResponseWriter adapts a Fiber context to http.ResponseWriter so that
// a standard net/http handler can write its response through Fiber.
type fiberResponseWriter struct {
	ctx           fiber.Ctx
	header        http.Header
	statusCode    int
	headerWritten bool
}

func newFiberResponseWriter(ctx fiber.Ctx) *fiberResponseWriter {
	return &fiberResponseWriter{ctx: ctx}
}

// Header returns the response header map. Lazily initialized.
func (w *fiberResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

// WriteHeader sends an HTTP response header with the provided status code.
// Uses Add (not Set) to preserve multi-value headers like Set-Cookie.
// Calling WriteHeader more than once is a no-op.
func (w *fiberResponseWriter) WriteHeader(code int) {
	if w.headerWritten {
		return
	}
	w.headerWritten = true
	w.statusCode = code

	for k, vals := range w.header {
		for _, v := range vals {
			w.ctx.Response().Header.Add(k, v)
		}
	}
	w.ctx.Status(code)
}

// Write writes the data to the Fiber response body. If WriteHeader has not
// been called, it calls WriteHeader(http.StatusOK) before writing.
func (w *fiberResponseWriter) Write(data []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ctx.Write(data)
}

// Flush implements http.Flusher. This is a no-op for Fiber since the
// response is buffered and sent after the handler returns.
func (w *fiberResponseWriter) Flush() {}
