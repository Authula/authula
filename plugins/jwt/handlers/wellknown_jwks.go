package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/usecases"
)

type WellKnownJWKSHandler struct {
	Logger      models.Logger
	JWKSUseCase usecases.JWKSUseCase
}

func (h *WellKnownJWKSHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		result, err := h.JWKSUseCase.GetJWKS(ctx)
		if err != nil {
			h.Logger.Error("failed to fetch JWKS", "error", err)
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]string{
				"error": "failed to fetch JWKS",
			})
			reqCtx.Handled = true
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=3600")
		reqCtx.SetJSONResponse(http.StatusOK, result.KeySet)
	}
}
