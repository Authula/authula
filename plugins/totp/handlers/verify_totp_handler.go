package handlers

import (
	"errors"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/usecases"
)

type VerifyTOTPHandler struct {
	PluginConfig *types.TOTPPluginConfig
	UseCase      *usecases.VerifyTOTPUseCase
}

func (h *VerifyTOTPHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		cookie, err := r.Cookie(constants.CookieTOTPPending)
		if err != nil || cookie.Value == "" {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "invalid or expired pending token",
			})
			reqCtx.Handled = true
			return
		}

		var payload types.VerifyTOTPRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		clientIP := reqCtx.ClientIP
		userAgent := r.UserAgent()

		result, err := h.UseCase.Verify(ctx, cookie.Value, payload.Code, payload.TrustDevice, &clientIP, &userAgent)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, constants.ErrInvalidTOTPCode) || errors.Is(err, constants.ErrTOTPNotEnabled) || errors.Is(err, constants.ErrInvalidPendingToken) {
				status = http.StatusUnauthorized
			}
			reqCtx.SetJSONResponse(status, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		reqCtx.SetUserIDInContext(result.User.ID)
		reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
		reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
		reqCtx.Values[models.ContextAuthSuccess.String()] = true

		if result.TrustedDeviceToken != "" {
			http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
				Name:     constants.CookieTOTPTrusted,
				Value:    result.TrustedDeviceToken,
				Path:     "/",
				MaxAge:   int(h.PluginConfig.TrustedDeviceDuration.Seconds()),
				HttpOnly: true,
				Secure:   h.PluginConfig.SecureCookie,
				SameSite: types.ParseSameSite(h.PluginConfig.SameSite),
			})
		}

		http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
			Name:     constants.CookieTOTPPending,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.PluginConfig.SecureCookie,
			SameSite: types.ParseSameSite(h.PluginConfig.SameSite),
		})

		reqCtx.SetJSONResponse(http.StatusOK, &types.VerifyTOTPResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
