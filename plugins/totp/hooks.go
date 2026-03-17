package totp

import (
	"net/http"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

type TOTPHookID string

const (
	HookIDTOTPIntercept TOTPHookID = "totp.intercept"
)

func (id TOTPHookID) String() string {
	return string(id)
}

func (p *TOTPPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:    models.HookAfter,
			PluginID: HookIDTOTPIntercept.String(),
			Matcher:  p.signInSuccessMatcher,
			Handler:  p.interceptSignInHook,
			Order:    1, // Must run before session (5) and JWT (10) hooks to intercept auth
		},
	}
}

func (p *TOTPPlugin) signInSuccessMatcher(reqCtx *models.RequestContext) bool {
	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	return ok && authSuccess
}

func (p *TOTPPlugin) interceptSignInHook(reqCtx *models.RequestContext) error {
	// Skip totp verify routes to avoid intercepting our own auth success
	if strings.Contains(reqCtx.Request.URL.Path, "/totp/verify") {
		return nil
	}

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	userID := *reqCtx.UserID

	// Check if user has 2FA enabled via the totp table
	enabled, err := p.totpRepo.IsEnabled(reqCtx.Request.Context(), userID)
	if err != nil {
		p.logger.Error("failed to check totp status", map[string]any{"error": err.Error(), "user_id": userID})
		reqCtx.SetJSONResponse(http.StatusServiceUnavailable, map[string]any{
			"message": "totp verification unavailable",
		})
		reqCtx.Handled = true
		return nil
	}
	if !enabled {
		return nil // No 2FA, pass through
	}

	// Check for trusted device cookie
	if p.hasTrustedDevice(reqCtx, userID) {
		return nil // Trusted device, pass through
	}

	// Create pending verification token
	token, err := p.tokenService.Generate()
	if err != nil {
		p.logger.Error("failed to generate pending token", map[string]any{"error": err.Error()})
		reqCtx.SetJSONResponse(http.StatusServiceUnavailable, map[string]any{
			"message": "totp verification unavailable",
		})
		reqCtx.Handled = true
		return nil
	}
	hashedToken := p.tokenService.Hash(token)

	_, err = p.verificationService.Create(
		reqCtx.Request.Context(),
		userID,
		hashedToken,
		models.TypeTOTPPendingAuth,
		userID, // identifier
		p.pluginConfig.PendingTokenExpiry,
	)
	if err != nil {
		p.logger.Error("failed to create pending verification", map[string]any{"error": err.Error()})
		reqCtx.SetJSONResponse(http.StatusServiceUnavailable, map[string]any{
			"message": "totp verification unavailable",
		})
		reqCtx.Handled = true
		return nil
	}

	// Set pending token cookie
	http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
		Name:     constants.CookieTOTPPending,
		Value:    token,
		Path:     "/",
		MaxAge:   int(p.pluginConfig.PendingTokenExpiry.Seconds()),
		HttpOnly: true,
		Secure:   p.pluginConfig.SecureCookie,
		SameSite: types.ParseSameSite(p.pluginConfig.SameSite),
	})

	// Clear session values — prevent session creation
	delete(reqCtx.Values, models.ContextSessionID.String())
	delete(reqCtx.Values, models.ContextSessionToken.String())
	delete(reqCtx.Values, models.ContextAuthSuccess.String())

	// Replace response with totpRedirect
	reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
		"totpRedirect": true,
	})

	return nil
}

func (p *TOTPPlugin) hasTrustedDevice(reqCtx *models.RequestContext, userID string) bool {
	cookie, err := reqCtx.Request.Cookie(constants.CookieTOTPTrusted)
	if err != nil || cookie.Value == "" {
		return false
	}

	hashedToken := p.tokenService.Hash(cookie.Value)
	device, err := p.totpRepo.GetTrustedDeviceByToken(reqCtx.Request.Context(), hashedToken)
	if err != nil || device == nil {
		return false
	}

	if device.UserID != userID {
		return false
	}

	if device.ExpiresAt.Before(time.Now().UTC()) {
		return false
	}

	newExpiry := time.Now().UTC().Add(p.pluginConfig.TrustedDeviceDuration)
	_ = p.totpRepo.RefreshTrustedDevice(reqCtx.Request.Context(), hashedToken, newExpiry)

	return true
}
