package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/types"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/usecases"
)

type GetMeHandler struct {
	UseCase *usecases.GetMeUseCase
}

func (h *GetMeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "unauthorized",
			})
			reqCtx.Handled = true
			return
		}

		result, err := h.UseCase.GetMe(ctx, userID)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetMeResponse{
			User: result.User,
		})
	}
}
