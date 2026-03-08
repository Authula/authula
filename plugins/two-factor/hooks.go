package twofactor

import (
	"net/http"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
)

func (p *TwoFactorPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookAfter,
			Matcher: p.signInSuccessMatcher,
			Handler: p.interceptSignInHook,
			Order:   1, // Must run before session (5) and JWT (10) hooks to intercept auth
		},
	}
}

func (p *TwoFactorPlugin) signInSuccessMatcher(reqCtx *models.RequestContext) bool {
	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	return ok && authSuccess
}

func (p *TwoFactorPlugin) interceptSignInHook(reqCtx *models.RequestContext) error {
	// Skip two-factor verify routes to avoid intercepting our own auth success
	if strings.Contains(reqCtx.Request.URL.Path, "/two-factor/verify") {
		return nil
	}

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	userID := *reqCtx.UserID

	// Check if user has 2FA enabled via the two_factor table
	enabled, err := p.twoFactorRepo.IsEnabled(reqCtx.Request.Context(), userID)
	if err != nil || !enabled {
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
		Secure:   p.pluginConfig.SecureCookie,
		SameSite: types.ParseSameSite(p.pluginConfig.SameSite),
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
	device, err := p.twoFactorRepo.GetTrustedDeviceByToken(reqCtx.Request.Context(), hashedToken)
	if err != nil || device == nil {
		return false
	}

	// Check expiry
	if device.ExpiresAt.Before(time.Now().UTC()) {
		return false
	}

	// Refresh expiry
	newExpiry := time.Now().UTC().Add(p.pluginConfig.TrustedDeviceDuration)
	_ = p.twoFactorRepo.RefreshTrustedDevice(reqCtx.Request.Context(), hashedToken, newExpiry)

	return true
}
