package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/types"
	"github.com/Authula/authula/plugins/totp/usecases"
)

type GetTOTPURIHandler struct {
	GlobalConfig *models.Config
	UseCase      *usecases.GetTOTPURIUseCase
}

func (h *GetTOTPURIHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		principal, ok := models.GetActorFromContext(ctx)
		if !ok || principal == nil || principal.ID == "" {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "Unauthorized",
			})
			reqCtx.Handled = true
			return
		}

		totpURI, err := h.UseCase.GetTOTPURI(ctx, principal.ID, h.GlobalConfig.AppName)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetTOTPURIResponse{
			TotpURI: totpURI,
		})
	}
}
