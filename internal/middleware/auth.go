package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"slices"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type AuthContextKey string

const ContextUserID AuthContextKey = "user_id"

// getUserIDFromCookie extracts the user ID from the session cookie.
// Returns an error if the cookie is missing, invalid, or session is not found.
func getUserIDFromCookie(authService *auth.Service, cookieName string, r *http.Request) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return "", err
	}

	sess, err := authService.SessionService.GetSessionByToken(authService.TokenService.HashToken(cookie.Value))
	if err != nil || sess == nil {
		return "", err
	}

	return sess.UserID, nil
}

// validateCSRF checks the CSRF token from cookie and header.
// Returns an error if validation fails.
func validateCSRF(csrfConfig models.CSRFConfig, r *http.Request) error {
	if !csrfConfig.Enabled {
		return nil
	}

	cookie, err := r.Cookie(csrfConfig.CookieName)
	if err != nil {
		return err
	}

	header := r.Header.Get(csrfConfig.HeaderName)
	if header == "" {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
		return err
	}

	return nil
}

func AuthMiddleware(authService *auth.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromCookie(authService, cookieName, r)
			if err != nil {
				util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuthMiddleware(authService *auth.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, err := getUserIDFromCookie(authService, cookieName, r); err == nil {
				ctx := context.WithValue(r.Context(), ContextUserID, userID)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CorsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
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
				// Only respond with 204 if origin is allowed, otherwise respond with 403
				if !allowed && origin != "" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CSRFMiddleware(csrfConfig models.CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet ||
				r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if err := validateCSRF(csrfConfig, r); err != nil {
				util.JSONResponse(w, http.StatusForbidden, map[string]any{"message": "invalid CSRF token"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RedirectAuthMiddleware redirects unauthenticated users to the specified URL.
func RedirectAuthMiddleware(authService *auth.Service, cookieName string, redirectURL string, status int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromCookie(authService, cookieName, r)
			if err != nil || userID == "" {
				http.Redirect(w, r, redirectURL, status)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
