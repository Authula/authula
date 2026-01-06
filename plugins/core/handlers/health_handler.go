package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/usecases"
)

type HealthHandler struct {
	UseCase usecases.HealthCheckUseCase
}

func (h *HealthHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		result, err := h.UseCase.HealthCheck(ctx)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"error": err.Error(),
			})
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, result)
	}
}
