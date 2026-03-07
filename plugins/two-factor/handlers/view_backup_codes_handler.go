package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/usecases"
)

type ViewBackupCodesHandler struct {
	UseCase usecases.ViewBackupCodesUseCase
}

func (h *ViewBackupCodesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		if reqCtx.UserID == nil || *reqCtx.UserID == "" {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "authentication required",
			})
			reqCtx.Handled = true
			return
		}

		var payload types.ViewBackupCodesRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		count, err := h.UseCase.View(ctx, *reqCtx.UserID, payload.Password)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.ViewBackupCodesResponse{
			RemainingCount: count,
		})
	}
}
