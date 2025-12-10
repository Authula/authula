package middleware

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
)

func RateLimitMiddleware(rateLimitService *auth.RateLimitService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			clientIP := rateLimitService.GetClientIP(req)

			allowed, err := rateLimitService.Allow(ctx, clientIP, req)
			if err != nil {
				util.JSONResponse(w, http.StatusInternalServerError, map[string]any{"message": "rate-limit error"})
				return
			}
			if !allowed {
				util.JSONResponse(w, http.StatusTooManyRequests, map[string]any{"message": "rate limit exceeded"})
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}
