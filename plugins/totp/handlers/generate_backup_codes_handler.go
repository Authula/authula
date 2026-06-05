package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/types"
	"github.com/Authula/authula/plugins/totp/usecases"
)

type GenerateBackupCodesHandler struct {
	UseCase *usecases.GenerateBackupCodesUseCase
}

func (h *GenerateBackupCodesHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		actor, ok := models.GetActorFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "Unauthorized",
			})
			reqCtx.Handled = true
			return
		}

		codes, err := h.UseCase.Generate(ctx, actor.ID)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GenerateBackupCodesResponse{
			BackupCodes: codes,
		})
	}
}
