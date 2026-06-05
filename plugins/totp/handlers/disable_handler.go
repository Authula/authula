package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/usecases"
)

type DisableHandler struct {
	UseCase *usecases.DisableUseCase
}

func (h *DisableHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if err := h.UseCase.Disable(ctx, reqCtx.Actor.ID); err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
			"message": "totp authentication disabled",
		})
	}
}
