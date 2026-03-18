package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/usecases"
)

type EnableHandler struct {
	UseCase      *usecases.EnableUseCase
	PluginConfig *types.TOTPPluginConfig
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

		issuer := ""
		if r.Body != nil {
			var payload types.EnableRequest
			if err := util.ParseJSON(r, &payload); err != nil {
				if !errors.Is(err, io.EOF) {
					reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
						"message": "invalid request body",
					})
					reqCtx.Handled = true
					return
				}
			} else {
				issuer = payload.Issuer
			}
		}

		result, err := h.UseCase.Enable(ctx, userID, issuer)
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
