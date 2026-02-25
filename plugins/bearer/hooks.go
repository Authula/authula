package bearer

import (
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type BearerHookID string

const (
	HookIDBearerAuth         BearerHookID = "bearer.auth"
	HookIDBearerAuthOptional BearerHookID = "bearer.auth.optional"
)

func (id BearerHookID) String() string {
	return string(id)
}

func (p *BearerPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:    models.HookBefore,
			PluginID: HookIDBearerAuth.String(),
			Handler:  p.validateBearerToken,
			Order:    10,
		},
		{
			Stage:    models.HookBefore,
			PluginID: HookIDBearerAuthOptional.String(),
			Handler:  p.validateBearerTokenOptional,
			Order:    10,
		},
	}
}

func (p *BearerPlugin) validateBearerToken(reqCtx *models.RequestContext) error {
	// Cooperative auth: if UserID already set by another auth plugin, skip
	if reqCtx.UserID != nil {
		return nil
	}

	token, err := p.extractToken(reqCtx.Request)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
			"message": err.Error(),
		})
		reqCtx.Handled = true
		return nil
	}

	userID, err := p.jwtService.ValidateToken(token)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{
			"message": "Bearer token invalid or expired",
		})
		reqCtx.Handled = true
		return nil
	}

	reqCtx.SetUserIDInContext(userID)

	return nil
}

func (p *BearerPlugin) validateBearerTokenOptional(reqCtx *models.RequestContext) error {
	// Cooperative auth: if UserID already set by another auth plugin, skip
	if reqCtx.UserID != nil {
		return nil
	}

	token, err := p.extractToken(reqCtx.Request)
	if err != nil {
		// No token, skip silently
		return nil
	}

	userID, err := p.jwtService.ValidateToken(token)
	if err != nil {
		return nil
	}

	reqCtx.SetUserIDInContext(userID)

	return nil
}
