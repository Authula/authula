package middleware

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/Authula/authula/models"
)

func RequireActor(types ...models.ActorType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCtx, _ := models.GetRequestContext(r.Context())

			if reqCtx.Actor == nil {
				reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
					"message": "unauthorized",
				})
				reqCtx.Handled = true
				return
			}

			if slices.Contains(types, reqCtx.Actor.Type) {
				next.ServeHTTP(w, r)
				return
			}

			reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{
				"message": fmt.Sprintf("only %v actors are allowed for this action", types),
			})
			reqCtx.Handled = true
		})
	}
}

func RequirePublicOrUserActor() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCtx, _ := models.GetRequestContext(r.Context())

			if reqCtx.Actor == nil || reqCtx.Actor.Type == models.ActorUser {
				next.ServeHTTP(w, r)
				return
			}

			reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{
				"message": "only public or user actors are allowed for this action",
			})
			reqCtx.Handled = true
		})
	}
}
