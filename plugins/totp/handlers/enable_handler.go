package handlers

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/usecases"
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

		userID, ok := models.GetUserIDFromContext(ctx)
		if !ok {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "Unauthorized",
			})
			reqCtx.Handled = true
			return
		}

		result, err := h.UseCase.Enable(ctx, userID, h.GlobalConfig.AppName)
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
