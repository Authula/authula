package handlers

import (
	"net/http"

	"github.com/Authula/authula/internal/types"
	"github.com/Authula/authula/internal/usecases"
	"github.com/Authula/authula/models"
)

type GetMeHandler struct {
	UseCase *usecases.GetMeUseCase
}

func (h *GetMeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		result, err := h.UseCase.GetMe(ctx, reqCtx.Actor.ID)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetMeResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
