package middleware

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
)

// AdminAuth is middleware that checks for a valid admin API key in the request headers.
func AdminAuth(apiKey string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-KEY")
			if key != apiKey || apiKey == "" {
				util.JSONResponse(w, http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
