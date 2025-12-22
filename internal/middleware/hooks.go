package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// responseBufferWriter captures the response for inspection and modification.
type responseBufferWriter struct {
	status      int
	header      http.Header
	body        bytes.Buffer
	cookies     []*http.Cookie
	wroteHeader bool
}

func newResponseBufferWriter() *responseBufferWriter {
	return &responseBufferWriter{
		header:  make(http.Header),
		cookies: []*http.Cookie{},
	}
}

func (rw *responseBufferWriter) Header() http.Header {
	return rw.header
}

func (rw *responseBufferWriter) WriteHeader(statusCode int) {
	if !rw.wroteHeader {
		rw.status = statusCode
		rw.wroteHeader = true
	}
}

func (rw *responseBufferWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.body.Write(b)
}

func (rw *responseBufferWriter) SetCookie(cookie *http.Cookie) {
	rw.cookies = append(rw.cookies, cookie)
}

// Redirect sets up a redirect response and marks the request as handled.
func Redirect(ctx *models.EndpointHookContext, url string, status int) {
	if status == 0 {
		status = http.StatusSeeOther // Default to 303 See Other
	}
	ctx.ResponseStatus = status
	ctx.ResponseHeaders = map[string][]string{"Location": {url}}
	ctx.ResponseBody = nil
	ctx.ResponseCookies = []*http.Cookie{}
	ctx.Handled = true
}

// writeResponseFromContext writes the response using the provided properties.
func writeResponseFromContext(w http.ResponseWriter, status int, headers map[string][]string, body []byte, cookies []*http.Cookie) {
	for k, v := range headers {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	for _, cookie := range cookies {
		http.SetCookie(w, cookie)
	}
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if body != nil {
		w.Write(body)
	}
}

func EndpointHooksMiddleware(config *models.Config, authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.EndpointHooks.Before == nil && config.EndpointHooks.After == nil && config.EndpointHooks.Response == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Always buffer to allow consistent modifications
			buf := newResponseBufferWriter()

			hookCtx := &models.EndpointHookContext{
				Path:            r.URL.Path,
				Method:          r.Method,
				Headers:         make(map[string][]string),
				Query:           make(map[string][]string),
				Request:         r,
				ResponseStatus:  0,
				ResponseHeaders: make(map[string][]string),
				ResponseBody:    nil,
				ResponseCookies: []*http.Cookie{},
			}
			hookCtx.Redirect = func(url string, s int) {
				Redirect(hookCtx, url, s)
			}

			// Populate hookCtx
			for k, v := range r.Header {
				if len(v) > 0 {
					hookCtx.Headers[k] = v
				}
			}
			hookCtx.Query = r.URL.Query()

			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					r.GetBody = func() (io.ReadCloser, error) {
						return io.NopCloser(bytes.NewReader(bodyBytes)), nil
					}
					if len(bodyBytes) > 0 {
						var bodyMap map[string]any
						if json.Unmarshal(bodyBytes, &bodyMap) == nil {
							hookCtx.Body = bodyMap
						}
					}
				}
			}

			cookie, err := r.Cookie(config.Session.CookieName)
			if err == nil && cookie.Value != "" {
				session, _ := authService.SessionService.GetSessionByToken(authService.TokenService.HashToken(cookie.Value))
				if session != nil {
					user, _ := authService.UserService.GetUserByID(session.UserID)
					if user != nil {
						hookCtx.User = user
					}
				}
			}

			if config.EndpointHooks.Before != nil {
				if err := config.EndpointHooks.Before(hookCtx); err != nil {
					util.JSONResponse(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
					return
				}

				if hookCtx.Handled {
					writeResponseFromContext(
						w,
						hookCtx.ResponseStatus,
						hookCtx.ResponseHeaders,
						hookCtx.ResponseBody,
						hookCtx.ResponseCookies,
					)
					return
				}
			}

			// Add headers from Before to buf
			for k, v := range hookCtx.ResponseHeaders {
				for _, vv := range v {
					buf.Header().Add(k, vv)
				}
			}

			// Call handler
			next.ServeHTTP(buf, r)

			// Set response data from buffer
			hookCtx.ResponseStatus = buf.status
			hookCtx.ResponseBody = buf.body.Bytes()
			hookCtx.ResponseHeaders = make(map[string][]string)
			for k, v := range buf.header {
				hookCtx.ResponseHeaders[k] = slices.Clone(v)
			}
			hookCtx.ResponseCookies = append(hookCtx.ResponseCookies, buf.cookies...)

			// Response hook
			if config.EndpointHooks.Response != nil {
				if err := config.EndpointHooks.Response(hookCtx); err != nil {
					slog.Error("Error in Response Hook", "path", hookCtx.Path, "error", err)
					util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "Internal Server Error"})
					return
				}
			}

			// Write final response
			writeResponseFromContext(
				w,
				hookCtx.ResponseStatus,
				hookCtx.ResponseHeaders,
				hookCtx.ResponseBody,
				hookCtx.ResponseCookies,
			)

			// After hook
			if config.EndpointHooks.After != nil {
				config.EndpointHooks.After(hookCtx)
			}
		})
	}
}
