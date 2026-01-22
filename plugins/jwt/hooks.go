package jwt

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// issueTokensHook generates and stores JWT tokens for authenticated users
// This hook runs at HookAfter stage if "jwt.issuance" is in route.Metadata["plugins"]
// Expects ctx.Values["user_id"] and ctx.Values["session_id"] to be set
func (p *JWTPlugin) issueTokensHook(reqCtx *models.RequestContext) error {
	p.Logger.Debug("jwt issuance hook running", "path", reqCtx.Path)

	if reqCtx.UserID == nil {
		return nil
	}

	sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string)
	if !ok || sessionID == "" {
		return nil
	}

	tokenPair, err := p.GenerateTokens(p.globalConfig.Secret, *reqCtx.UserID, sessionID)
	if err != nil {
		p.Logger.Error("failed to generate JWT tokens", "user_id", *reqCtx.UserID, "session_id", sessionID, "error", err)
		// Return error to fail the request - JWT generation should not silently fail
		return fmt.Errorf("failed to generate authentication tokens: %w", err)
	}

	// Store tokens in context for response handling
	reqCtx.Values["access_token"] = tokenPair.AccessToken
	reqCtx.Values["refresh_token"] = tokenPair.RefreshToken
	p.Logger.Debug("access token generated", "access_token", tokenPair.AccessToken)
	p.Logger.Debug("refresh token generated", "refresh_token", tokenPair.RefreshToken)

	p.Logger.Debug("jwt tokens generated successfully", "user_id", *reqCtx.UserID)

	return nil
}

// buildHooks returns the configured hooks for this plugin
// Uses the new PluginID-based hook filtering for metadata-driven execution
func (p *JWTPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// JWT issuance hook: generates access and refresh tokens after authentication
		{
			Stage: models.HookAfter,
			Matcher: func(reqCtx *models.RequestContext) bool {
				sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string)
				return ok && sessionID != "" && reqCtx.UserID != nil
			},
			Handler: p.issueTokensHook,
			Order:   10, // Normal priority
		},
	}
}
