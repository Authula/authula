package admin

import (
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func (p *AdminPlugin) Hooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookBefore,
			Handler: p.enforceState,
			Order:   15,
		},
	}
}

func (p *AdminPlugin) enforceState(reqCtx *models.RequestContext) error {
	if reqCtx == nil || reqCtx.Request == nil {
		return nil
	}

	if reqCtx.UserID == nil || *reqCtx.UserID == "" {
		return nil
	}

	ctx := reqCtx.Request.Context()

	state, err := p.Api.GetUserState(ctx, *reqCtx.UserID)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to evaluate user state"})
		reqCtx.Handled = true
		return nil
	}

	if state != nil && state.IsBanned {
		if state.BannedUntil == nil || state.BannedUntil.After(time.Now().UTC()) {
			reqCtx.SetJSONResponse(http.StatusForbidden, map[string]any{"message": "user is banned"})
			reqCtx.Handled = true
			return nil
		}
	}

	rawSessionID, hasSessionID := reqCtx.Values[models.ContextSessionID.String()]
	if !hasSessionID || rawSessionID == nil {
		return nil
	}

	sessionID, ok := rawSessionID.(string)
	if !ok || sessionID == "" {
		return nil
	}

	sessionState, err := p.Api.GetSessionState(ctx, sessionID)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to evaluate session state"})
		reqCtx.Handled = true
		return nil
	}

	if sessionState != nil && sessionState.RevokedAt != nil {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "session is revoked"})
		reqCtx.Handled = true
		return nil
	}

	return nil
}
