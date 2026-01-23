package router

import (
	"net/http"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// DeferredResponseWriter buffers response writes to allow hooks to modify headers after handler execution.
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

	w.writeDeferred = true
	w.buffer = append(w.buffer, b...)
	return len(b), nil
}

// Flush writes all buffered data to the underlying writer
func (w *DeferredResponseWriter) Flush() error {
	if w.override {
		// Preserve existing cookies set by previous hooks
		existingCookies := w.Wrapped.Header()["Set-Cookie"]

		// Apply override headers to wrapped writer (except Set-Cookie, which we merge)
		if w.overrideHeaders != nil {
			for key, values := range w.overrideHeaders {
				if key != "Set-Cookie" {
					w.Wrapped.Header()[key] = append([]string(nil), values...)
				}
			}
		}

		// Merge cookies: existing (from previous hooks) + override (from context response)
		allCookies := make([]string, 0, len(existingCookies))
		allCookies = append(allCookies, existingCookies...)

		if w.overrideHeaders != nil {
			overrideCookies := w.overrideHeaders["Set-Cookie"]
			for _, overrideCookie := range overrideCookies {
				cookieName := extractCookieName(overrideCookie)
				// Only add if not already present (no duplicate cookie names)
				isDuplicate := false
				for _, existing := range existingCookies {
					if extractCookieName(existing) == cookieName {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					allCookies = append(allCookies, overrideCookie)
				}
			}
		}

		// Set all cookies on the wrapped writer
		w.Wrapped.Header()["Set-Cookie"] = allCookies

		// Write status and body
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

	// Normal response (no override)
	if w.headerWritten {
		w.Wrapped.WriteHeader(w.statusCode)
	}
	if w.writeDeferred && len(w.buffer) > 0 {
		_, err := w.Wrapped.Write(w.buffer)
		return err
	}
	return nil
}

// extractCookieName extracts the cookie name from a Set-Cookie header string
func extractCookieName(cookieHeader string) string {
	if before, _, ok := strings.Cut(cookieHeader, "="); ok {
		return before
	}
	return cookieHeader
}
