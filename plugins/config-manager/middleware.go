package configmanager

import (
	"net/http"
	"os"

	"github.com/GoBetterAuth/go-better-auth/v2/env"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
)

func ConfigManagerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-KEY")
		expectedKey := os.Getenv(env.EnvAdminApiKey)
		if apiKey != expectedKey || apiKey == "" {
			util.JSONResponse(w, http.StatusUnauthorized, map[string]any{
				"message": "missing or invalid API key.",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
