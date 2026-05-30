package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/usecases"
)

type WellKnownJWKSHandler struct {
	Logger      models.Logger
	JWKSUseCase usecases.JWKSUseCase
}

func (h *WellKnownJWKSHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		result, err := h.JWKSUseCase.GetJWKS(ctx)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=3600")
		reqCtx.SetJSONResponse(http.StatusOK, result.KeySet)
	}
}
