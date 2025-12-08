package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"slices"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type ctxKey string

const ContextUserID ctxKey = "user_id"

func AuthMiddleware(authService *auth.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
				return
			}

			sess, err := authService.SessionService.GetSessionByToken(authService.TokenService.HashToken(cookie.Value))
			if err != nil || sess == nil {
				util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "invalid session"})
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, sess.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuthMiddleware(authService *auth.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err == nil && cookie.Value != "" {
				if sess, _ := authService.SessionService.GetSessionByToken(authService.TokenService.HashToken(cookie.Value)); sess != nil {
					r = r.WithContext(context.WithValue(r.Context(), ContextUserID, sess.UserID))
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CorsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := slices.Contains(allowedOrigins, origin)

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,Cookie,Set-Cookie")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CSRFMiddleware(csrfConfig domain.CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet ||
				r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if !csrfConfig.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(csrfConfig.CookieName)
			if err != nil {
				util.JSONResponse(w, http.StatusForbidden, map[string]any{"message": "missing CSRF cookie"})
				return
			}

			header := r.Header.Get(csrfConfig.HeaderName)
			if header == "" {
				util.JSONResponse(w, http.StatusForbidden, map[string]any{"message": "missing CSRF header"})
				return
			}

			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				util.JSONResponse(w, http.StatusForbidden, map[string]any{"message": "invalid CSRF token"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
