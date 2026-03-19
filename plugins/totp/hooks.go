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
			// Must run before hooks that set auth cookies/sessions/tokens to intercept before those are set
			// preventing them from being issued until after TOTP verification.
			Order: 5,
		},
	}
}

func (p *TOTPPlugin) signInSuccessMatcher(reqCtx *models.RequestContext) bool {
	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	return ok && authSuccess
}

func (p *TOTPPlugin) interceptSignInHook(reqCtx *models.RequestContext) error {
	ctx := reqCtx.Request.Context()

	// Skip totp verify routes to avoid intercepting our own auth success
	if strings.Contains(reqCtx.Path, "/totp/verify") {
		return nil
	}

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}
	userID := *reqCtx.UserID

	enabled, err := p.totpRepo.IsEnabled(ctx, userID)
	if err != nil {
		p.logger.Error("failed to check totp status", map[string]any{"error": err.Error(), "user_id": userID})
		reqCtx.SetJSONResponse(http.StatusServiceUnavailable, map[string]any{
			"message": "totp verification unavailable",
		})
		reqCtx.Handled = true
		return nil
	}
	if !enabled {
		return nil // TOTP not enabled, pass through
	}

	if p.hasTrustedDevice(reqCtx, userID) {
		return nil // Trusted device, pass through
	}

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
		ctx,
		userID,
		hashedToken,
		models.TypeTOTPPendingAuth,
		userID,
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

	reqCtx.SetJSONResponse(http.StatusOK, &types.TOTPRedirectResponse{
		TOTPRedirect: true,
	})

	return nil
}

func (p *TOTPPlugin) hasTrustedDevice(reqCtx *models.RequestContext, userID string) bool {
	ctx := reqCtx.Request.Context()

	cookie, err := reqCtx.Request.Cookie(constants.CookieTOTPTrusted)
	if err != nil || cookie.Value == "" {
		return false
	}

	hashedToken := p.tokenService.Hash(cookie.Value)
	device, err := p.totpRepo.GetTrustedDeviceByToken(ctx, hashedToken)
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
	_ = p.totpRepo.RefreshTrustedDevice(ctx, hashedToken, newExpiry)

	return true
}
