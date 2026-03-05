package twofactor

import (
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func (p *TwoFactorPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookAfter,
			Matcher: p.signInSuccessMatcher,
			Handler: p.interceptSignInHook,
			Order:   10,
		},
	}
}

func (p *TwoFactorPlugin) signInSuccessMatcher(reqCtx *models.RequestContext) bool {
	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	return ok && authSuccess
}

func (p *TwoFactorPlugin) interceptSignInHook(reqCtx *models.RequestContext) error {
	userID, ok := reqCtx.Values[models.ContextUserID.String()].(string)
	if !ok || userID == "" {
		return nil
	}

	// Check if user has 2FA enabled
	user, err := p.userService.GetByID(reqCtx.Request.Context(), userID)
	if err != nil || user == nil || user.TwoFactorEnabled == nil || !*user.TwoFactorEnabled {
		return nil // No 2FA, pass through
	}

	// Check for trusted device cookie
	if p.hasTrustedDevice(reqCtx) {
		return nil // Trusted device, pass through
	}

	// Create pending verification token
	token, err := p.tokenService.Generate()
	if err != nil {
		p.logger.Error("failed to generate pending token", map[string]any{"error": err.Error()})
		return nil
	}
	hashedToken := p.tokenService.Hash(token)

	_, err = p.verificationService.Create(
		reqCtx.Request.Context(),
		userID,
		hashedToken,
		models.TypeTwoFactorPendingAuth,
		userID, // identifier
		p.pluginConfig.PendingTokenExpiry,
	)
	if err != nil {
		p.logger.Error("failed to create pending verification", map[string]any{"error": err.Error()})
		return nil
	}

	// Set pending token cookie
	http.SetCookie(reqCtx.ResponseWriter, &http.Cookie{
		Name:     "two_factor_pending",
		Value:    token,
		Path:     "/",
		MaxAge:   int(p.pluginConfig.PendingTokenExpiry.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear session values — prevent session creation
	delete(reqCtx.Values, models.ContextSessionID.String())
	delete(reqCtx.Values, models.ContextSessionToken.String())
	delete(reqCtx.Values, models.ContextAuthSuccess.String())

	// Replace response with twoFactorRedirect
	reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
		"twoFactorRedirect": true,
	})

	return nil
}

func (p *TwoFactorPlugin) hasTrustedDevice(reqCtx *models.RequestContext) bool {
	cookie, err := reqCtx.Request.Cookie("two_factor_trusted")
	if err != nil || cookie.Value == "" {
		return false
	}

	hashedToken := p.tokenService.Hash(cookie.Value)
	device, err := p.repo.GetTrustedDeviceByToken(reqCtx.Request.Context(), hashedToken)
	if err != nil || device == nil {
		return false
	}

	// Check expiry
	if device.ExpiresAt.Before(time.Now()) {
		return false
	}

	// Refresh expiry
	newExpiry := time.Now().Add(p.pluginConfig.TrustedDeviceDuration)
	_ = p.repo.RefreshTrustedDevice(reqCtx.Request.Context(), hashedToken, newExpiry)

	return true
}
