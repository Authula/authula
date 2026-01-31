package jwt

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
)

type JWTHookID string

// Constants for jwt plugin hook IDs and metadata
const (
	// Responds with JWT tokens in HTTP Response after issuance
	HookIDJWTRespondJSON JWTHookID = "jwt.respond_json"
)

func (id JWTHookID) String() string {
	return string(id)
}

// issueTokensHook generates and stores JWT tokens for authenticated users
// This hook runs at HookAfter stage.
func (p *JWTPlugin) issueTokensHook(reqCtx *models.RequestContext) error {
	if reqCtx.UserID == nil {
		return nil
	}

	if skipMint, ok := reqCtx.Values[models.ContextAuthIdempotentSkipTokensMint.String()].(bool); ok && skipMint {
		return nil
	}

	sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string)
	if !ok || sessionID == "" {
		return nil
	}

	tokenPair, err := p.jwtService.GenerateTokens(context.Background(), *reqCtx.UserID, sessionID)
	if err != nil {
		p.Logger.Error("failed to generate JWT tokens", "user_id", *reqCtx.UserID, "session_id", sessionID, "error", err)
		// Return error to fail the request - JWT generation should not silently fail
		return fmt.Errorf("failed to generate authentication tokens: %w", err)
	}

	expiresAt := time.Now().Add(p.pluginConfig.RefreshExpiresIn)
	if err := p.refreshService.StoreInitialRefreshToken(reqCtx.Request.Context(), tokenPair.RefreshToken, sessionID, expiresAt); err != nil {
		p.Logger.Error("failed to store refresh token", "user_id", *reqCtx.UserID, "session_id", sessionID, "error", err)
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Store tokens in context for other hooks to handle sending response
	reqCtx.Values[types.JWTTokenTypeAccess.String()] = tokenPair.AccessToken
	reqCtx.Values[types.JWTTokenTypeRefresh.String()] = tokenPair.RefreshToken

	return nil
}

// respondHook hook sends the generated JWT tokens in the response
func (p *JWTPlugin) respondHook(reqCtx *models.RequestContext) error {
	if reqCtx.UserID == nil {
		return nil
	}

	access, ok1 := reqCtx.Values[types.JWTTokenTypeAccess.String()].(string)
	refresh, ok2 := reqCtx.Values[types.JWTTokenTypeRefresh.String()].(string)
	if !ok1 || !ok2 {
		return nil
	}

	reqCtx.SetJSONResponse(http.StatusOK, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
	})
	reqCtx.Handled = true

	return nil
}

// buildHooks returns the configured hooks for this plugin
func (p *JWTPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// JWT issuance hook: generates access and refresh tokens after authentication
		{
			Stage: models.HookAfter,
			Matcher: func(reqCtx *models.RequestContext) bool {
				authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
				return ok && authSuccess
			},
			Handler: p.issueTokensHook,
			Order:   10,
		},
		// JWT response hook: sends generated tokens in response
		{
			Stage:    models.HookOnResponse,
			PluginID: HookIDJWTRespondJSON.String(),
			Handler:  p.respondHook,
			Order:    10,
		},
	}
}
