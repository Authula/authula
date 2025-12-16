package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type responseBuffer struct {
	status  int
	header  map[string][]string
	body    bytes.Buffer
	written bool
}

func (w *responseBuffer) Header() http.Header {
	return w.header
}

func (w *responseBuffer) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(b)
}

func (w *responseBuffer) WriteHeader(statusCode int) {
	w.status = statusCode
	w.written = true
}

func EndpointHooksMiddleware(config *models.Config, authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.EndpointHooks.Before == nil && config.EndpointHooks.After == nil && config.EndpointHooks.Response == nil {
				next.ServeHTTP(w, r)
				return
			}

			hookCtx := &models.EndpointHookContext{
				Path:            r.URL.Path,
				Method:          r.Method,
				Headers:         make(map[string][]string),
				Query:           make(map[string][]string),
				Request:         r,
				ResponseHeaders: make(map[string][]string),
			}

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

				maps.Copy(w.Header(), hookCtx.ResponseHeaders)

				if len(hookCtx.ResponseCookies) > 0 {
					for _, cookie := range hookCtx.ResponseCookies {
						http.SetCookie(w, cookie)
					}
				}

				if hookCtx.ResponseStatus != 0 || hookCtx.ResponseBody != nil {
					status := hookCtx.ResponseStatus
					if status == 0 {
						status = http.StatusOK
					}
					w.WriteHeader(status)
					if hookCtx.ResponseBody != nil {
						w.Write(hookCtx.ResponseBody)
					}
					return
				}
			}

			if config.EndpointHooks.Response != nil {
				buf := &responseBuffer{
					header: make(map[string][]string),
				}
				next.ServeHTTP(buf, r)

				hookCtx.ResponseStatus = buf.status
				hookCtx.ResponseBody = buf.body.Bytes()
				for k, v := range buf.header {
					if len(v) > 0 {
						hookCtx.ResponseHeaders[k] = v
					}
				}

				if err := config.EndpointHooks.Response(hookCtx); err != nil {
					slog.Error("Error in Response Hook for %s: %v", hookCtx.Path, err)
					util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "Internal Server Error"})
					return
				}

				// Write final response
				maps.Copy(w.Header(), hookCtx.ResponseHeaders)

				if len(hookCtx.ResponseCookies) > 0 {
					for _, cookie := range hookCtx.ResponseCookies {
						http.SetCookie(w, cookie)
					}
				}

				status := hookCtx.ResponseStatus
				if status == 0 {
					status = http.StatusOK
				}
				w.WriteHeader(status)
				if hookCtx.ResponseBody != nil {
					w.Write(hookCtx.ResponseBody)
				}
			} else {
				next.ServeHTTP(w, r)
			}

			if config.EndpointHooks.After != nil {
				go func() {
					if err := config.EndpointHooks.After(hookCtx); err != nil {
						slog.Error("Error in After Hook for %s: %v", hookCtx.Path, err)
					}
				}()
			}
		})
	}
}
