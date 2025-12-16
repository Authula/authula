package middleware

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

func RateLimitMiddleware(rateLimitService models.RateLimitService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			clientIP := rateLimitService.GetClientIP(req)

			key := rateLimitService.BuildKey(clientIP)
			allowed, err := rateLimitService.Allow(ctx, key, req)
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
