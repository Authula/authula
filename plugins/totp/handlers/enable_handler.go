package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/constants"
	"github.com/Authula/authula/plugins/totp/types"
	"github.com/Authula/authula/plugins/totp/usecases"
)

type EnableHandler struct {
	GlobalConfig *models.Config
	PluginConfig *types.TOTPPluginConfig
	UseCase      *usecases.EnableUseCase
}

func (h *EnableHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		result, err := h.UseCase.Enable(ctx, reqCtx.Actor.ID, h.GlobalConfig.AppName)
		if err != nil {
			reqCtx.SetJSONResponse(http.StatusBadRequest, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		if result.PendingToken != "" {
			http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
				Name:     constants.CookieTOTPPending,
				Value:    result.PendingToken,
				Path:     "/",
				MaxAge:   int(h.PluginConfig.PendingTokenExpiry.Seconds()),
				HttpOnly: true,
				Secure:   h.PluginConfig.SecureCookie,
				SameSite: types.ParseSameSite(h.PluginConfig.SameSite),
			})
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.EnableResponse{
			TotpURI:     result.TotpURI,
			BackupCodes: result.BackupCodes,
		})
	}
}
