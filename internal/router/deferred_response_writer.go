package router

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// deferredResponseWriter buffers response writes to allow hooks to modify headers after handler execution.
// It has a maximum buffer size limit to prevent excessive memory usage.
// For streaming responses (Server-Sent Events, WebSockets), use http.Header() directly without buffering.
type DeferredResponseWriter struct {
	Wrapped         http.ResponseWriter
	Logger          models.Logger
	headerWritten   bool
	statusCode      int
	buffer          []byte
	writeDeferred   bool
	ctx             *models.RequestContext
	override        bool
	overrideStatus  int
	overrideBody    []byte
	overrideHeaders http.Header
	// MaxBufferSize limits the maximum amount of data that can be buffered
	MaxBufferSize int
	// SkipBuffer indicates that buffering should be skipped (for streaming responses)
	SkipBuffer bool
}

func (w *DeferredResponseWriter) SetRequestContext(ctx *models.RequestContext) {
	w.ctx = ctx
}

func (w *DeferredResponseWriter) GetRequestContext() *models.RequestContext {
	return w.ctx
}

func (w *DeferredResponseWriter) OverrideWithContext(ctx *models.RequestContext) {
	w.override = true
	w.overrideStatus = ctx.ResponseStatus
	if ctx.ResponseBody != nil {
		w.overrideBody = append([]byte(nil), ctx.ResponseBody...)
	} else {
		w.overrideBody = nil
	}
	if ctx.ResponseHeaders != nil {
		w.overrideHeaders = make(http.Header, len(ctx.ResponseHeaders))
		for key, values := range ctx.ResponseHeaders {
			w.overrideHeaders[key] = append([]string(nil), values...)
		}
	} else {
		w.overrideHeaders = nil
	}
}

func (w *DeferredResponseWriter) Header() http.Header {
	return w.Wrapped.Header()
}

func (w *DeferredResponseWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}
	w.statusCode = statusCode
	w.headerWritten = true
}

func (w *DeferredResponseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.statusCode = 200
		w.headerWritten = true
	}

	// If skipBuffer is set or streaming response detected, bypass buffering
	if w.SkipBuffer {
		return w.Wrapped.Write(b)
	}

	w.writeDeferred = true

	// Check if buffer would exceed max size
	if len(w.buffer)+len(b) > w.MaxBufferSize {
		w.Logger.Warn("Response buffer exceeded maximum size, bypassing buffering", "current", len(w.buffer), "requested", len(b), "max", w.MaxBufferSize)
		// Bypass buffering for large responses - flush immediately
		if len(w.buffer) > 0 {
			// Flush existing buffer first
			w.Wrapped.WriteHeader(w.statusCode)
			w.Wrapped.Write(w.buffer)
			w.buffer = nil
		}
		// Write new data directly without buffering
		w.SkipBuffer = true
		return w.Wrapped.Write(b)
	}

	w.buffer = append(w.buffer, b...)
	return len(b), nil
}

// Flush writes all buffered data to the underlying writer
func (w *DeferredResponseWriter) Flush() error {
	if w.override {
		if w.overrideHeaders != nil {
			headers := w.Wrapped.Header()
			for key, values := range w.overrideHeaders {
				headers[key] = append([]string(nil), values...)
			}
		}
		status := w.overrideStatus
		if status == 0 {
			if w.headerWritten {
				status = w.statusCode
			} else {
				status = http.StatusOK
			}
		}
		w.Wrapped.WriteHeader(status)
		if len(w.overrideBody) > 0 {
			_, err := w.Wrapped.Write(w.overrideBody)
			return err
		}
		return nil
	}
	if w.headerWritten {
		w.Wrapped.WriteHeader(w.statusCode)
	}
	if w.writeDeferred && len(w.buffer) > 0 {
		_, err := w.Wrapped.Write(w.buffer)
		return err
	}
	return nil
}
