package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/services"
)

type WellKnownJWKSHandler struct {
	CacheService services.CacheService
	Logger       models.Logger
}

func (h *WellKnownJWKSHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		jwkSet, err := h.CacheService.GetJWKSWithFallback(ctx)
		if err != nil {
			h.Logger.Error("failed to fetch JWKS", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "failed to fetch JWKS",
			})
			return
		}

		// Return JWKS as JSON
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		if err := json.NewEncoder(w).Encode(jwkSet); err != nil {
			h.Logger.Error("failed to encode JWKS", "error", err)
		}
	}
}
