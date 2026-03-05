package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/usecases"
)

type VerifyBackupCodeHandler struct {
	UseCase               usecases.VerifyBackupCodeUseCase
	VerificationService   VerificationService
	TokenService          TokenService
	TrustedDeviceDuration time.Duration
}

func (h *VerifyBackupCodeHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx, _ := models.GetRequestContext(ctx)

		// Read pending token from cookie
		cookie, err := r.Cookie("two_factor_pending")
		if err != nil || cookie.Value == "" {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "invalid or expired pending token",
			})
			reqCtx.Handled = true
			return
		}

		// Look up verification by hashed token
		hashedToken := h.TokenService.Hash(cookie.Value)
		verification, err := h.VerificationService.FindByToken(ctx, hashedToken, models.TypeTwoFactorPendingAuth)
		if err != nil || verification == nil || verification.UserID == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "invalid or expired pending token",
			})
			reqCtx.Handled = true
			return
		}

		// Check expiration
		if verification.ExpiresAt.Before(time.Now()) {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
				"message": "invalid or expired pending token",
			})
			reqCtx.Handled = true
			return
		}

		userID := *verification.UserID

		var payload types.VerifyBackupCodeRequest
		if err := util.ParseJSON(r, &payload); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{
				"message": "invalid request body",
			})
			reqCtx.Handled = true
			return
		}

		clientIP := reqCtx.ClientIP
		userAgent := r.UserAgent()

		result, err := h.UseCase.Verify(ctx, userID, payload.Code, payload.TrustDevice, &clientIP, &userAgent)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, constants.ErrInvalidBackupCode) || errors.Is(err, constants.ErrTwoFactorNotEnabled) {
				status = http.StatusUnauthorized
			}
			reqCtx.SetJSONResponse(status, map[string]any{
				"message": err.Error(),
			})
			reqCtx.Handled = true
			return
		}

		// Clean up the pending verification
		_ = h.VerificationService.Delete(ctx, verification.ID)

		// Set session context values
		reqCtx.SetUserIDInContext(result.User.ID)
		reqCtx.Values[models.ContextSessionID.String()] = result.Session.ID
		reqCtx.Values[models.ContextSessionToken.String()] = result.SessionToken
		reqCtx.Values[models.ContextAuthSuccess.String()] = true

		// Set trusted device cookie if applicable
		if result.TrustedDeviceToken != "" {
			http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
				Name:     "two_factor_trusted",
				Value:    result.TrustedDeviceToken,
				Path:     "/",
				MaxAge:   int(h.TrustedDeviceDuration.Seconds()),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
		}

		// Clear the pending cookie
		http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
			Name:     "two_factor_pending",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		reqCtx.SetJSONResponse(http.StatusOK, &types.VerifyBackupCodeResponse{
			User:    result.User,
			Session: result.Session,
		})
	}
}
