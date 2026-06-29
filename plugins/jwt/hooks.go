package jwt

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Authula/authula/models"
	jwtservices "github.com/Authula/authula/plugins/jwt/services"
	"github.com/Authula/authula/plugins/jwt/types"
)

type JWTHookID string

const (
	HookIDJWTRespondJSON JWTHookID = "jwt.respond_json"
)

func (id JWTHookID) String() string {
	return string(id)
}

func (p *JWTPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookAfter,
			Matcher: p.issueTokensHookMatcher,
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

func (p *JWTPlugin) issueTokensHookMatcher(reqCtx *models.RequestContext) bool {
	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	return ok && authSuccess
}

func (p *JWTPlugin) issueTokensHook(reqCtx *models.RequestContext) error {
	if reqCtx.Actor == nil {
		return nil
	}

	if skipMint, ok := reqCtx.Values[models.ContextAuthIdempotentSkipTokensMint.String()].(bool); ok && skipMint {
		return nil
	}

	ctx := reqCtx.Request.Context()

	switch reqCtx.Actor.Type {
	case models.ActorUser:
		{
			sessionID, ok := reqCtx.Values[models.ContextSessionID.String()].(string)
			if !ok || sessionID == "" {
				return nil
			}

			tokenPair, err := p.jwtService.(jwtservices.TokenService).GenerateUserToken(ctx, reqCtx.Actor.ID, sessionID)
			if err != nil {
				p.Logger.Error("failed to generate user JWT tokens", "user_id", reqCtx.Actor.ID, "session_id", sessionID, "error", err)
				return fmt.Errorf("failed to generate authentication tokens: %w", err)
			}

			expiresAt := time.Now().Add(p.pluginConfig.RefreshExpiresIn)
			if err := p.refreshService.StoreInitialRefreshToken(ctx, tokenPair.RefreshToken, sessionID, expiresAt); err != nil {
				p.Logger.Error("failed to store refresh token", "user_id", reqCtx.Actor.ID, "session_id", sessionID, "error", err)
				return fmt.Errorf("failed to store refresh token: %w", err)
			}

			reqCtx.Values[types.JWTTokenTypeAccess.String()] = tokenPair.AccessToken
			reqCtx.Values[types.JWTTokenTypeRefresh.String()] = tokenPair.RefreshToken
		}
	case models.ActorMachine:
		{
			orgID := ""
			orgID, ok := reqCtx.Actor.GetClaimString("organization_id")
			if !ok {
				return fmt.Errorf("ActorMachine has no org_id in its claims failing to be associated with an organization")
			}
			tokenPair, err := p.jwtService.(jwtservices.TokenService).GenerateMachineToken(
				ctx, reqCtx.Actor.ID, orgID, reqCtx.Actor.Scopes,
			)
			if err != nil {
				p.Logger.Error("failed to generate machine JWT token", "client_id", reqCtx.Actor.ID, "error", err)
				return fmt.Errorf("failed to generate machine authentication tokens: %w", err)
			}
			reqCtx.Values[types.JWTTokenTypeAccess.String()] = tokenPair.AccessToken
		}
	}

	return nil
}

func (p *JWTPlugin) respondHook(reqCtx *models.RequestContext) error {
	if reqCtx.Actor == nil {
		return nil
	}

	access, ok := reqCtx.Values[types.JWTTokenTypeAccess.String()].(string)
	if !ok || access == "" {
		return nil
	}

	payload := map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
	}

	if refresh, ok := reqCtx.Values[types.JWTTokenTypeRefresh.String()].(string); ok && refresh != "" {
		payload["refresh_token"] = refresh
	}

	reqCtx.SetJSONResponse(http.StatusOK, payload)
	reqCtx.Handled = true
	return nil
}
