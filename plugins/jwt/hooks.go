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

const (
	HookIDJWTRespondJSON JWTHookID = "jwt.respond_json"
)

func (id JWTHookID) String() string {
	return string(id)
}

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
		return fmt.Errorf("failed to generate authentication tokens: %w", err)
	}

	expiresAt := time.Now().Add(p.pluginConfig.RefreshExpiresIn)
	if err := p.refreshService.StoreInitialRefreshToken(reqCtx.Request.Context(), tokenPair.RefreshToken, sessionID, expiresAt); err != nil {
		p.Logger.Error("failed to store refresh token", "user_id", *reqCtx.UserID, "session_id", sessionID, "error", err)
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	reqCtx.Values[types.JWTTokenTypeAccess.String()] = tokenPair.AccessToken
	reqCtx.Values[types.JWTTokenTypeRefresh.String()] = tokenPair.RefreshToken

	return nil
}

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

func (p *JWTPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage: models.HookAfter,
			Matcher: func(reqCtx *models.RequestContext) bool {
				authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
				return ok && authSuccess
			},
			Handler: p.issueTokensHook,
			Order:   15,
		},
		{
			Stage:    models.HookOnResponse,
			PluginID: HookIDJWTRespondJSON.String(),
			Handler:  p.respondHook,
			Order:    10,
		},
	}
}
