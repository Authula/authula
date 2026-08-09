package admin

import (
	"net/http"
	"time"

	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
)

func (p *AdminPlugin) Hooks() []models.Hook {
	return []models.Hook{
		{
			Stage:   models.HookBefore,
			Handler: p.enforceState,
			Order:   15,
		},
		{
			Stage:   models.HookBefore,
			Handler: p.resolveImpersonationOriginalSession,
			Order:   17,
		},
		{
			Stage:   models.HookBefore,
			Handler: p.addImpersonationWhitelistScopes,
			Order:   25,
		},
	}
}

func (p *AdminPlugin) enforceState(reqCtx *models.RequestContext) error {
	if reqCtx.Actor == nil || reqCtx.Actor.ID == "" || reqCtx.Actor.Type != models.ActorUser {
		return nil
	}

	ctx := reqCtx.Request.Context()

	state, err := p.Api.GetSelfUserState(ctx, reqCtx.Actor)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to evaluate user state"})
		reqCtx.Handled = true
		return nil
	}

	if state != nil && state.Banned {
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

	sessionState, err := p.Api.GetSelfSessionState(ctx, sessionID)
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

func (p *AdminPlugin) resolveImpersonationOriginalSession(reqCtx *models.RequestContext) error {
	if reqCtx.Actor == nil || reqCtx.Actor.Type != models.ActorUser {
		return nil
	}

	// Check if impersonation claims are already present (JWT path)
	if _, hasID := reqCtx.Actor.Claims[adminconstants.ImpersonatorID]; hasID {
		return nil
	}

	cookieName := p.pluginCtx.GetConfig().Session.CookieName
	originalCookieName := cookieName + adminconstants.OriginalSessionCookieSuffix

	cookie, err := reqCtx.Request.Cookie(originalCookieName)
	if err != nil {
		return nil
	}

	rawSessionID, hasSessionID := reqCtx.Values[models.ContextSessionID.String()]
	if !hasSessionID || rawSessionID == nil {
		return nil
	}
	currentSessionID, ok := rawSessionID.(string)
	if !ok || currentSessionID == "" {
		return nil
	}

	// Verify the current session is an impersonation session via AdminSessionState
	sessionState, err := p.Api.SessionStateRepository().GetBySessionID(reqCtx.Request.Context(), currentSessionID)
	if err != nil || sessionState == nil || sessionState.ImpersonatorUserID == nil {
		return nil
	}

	hashedOriginal := p.tokenService.Hash(cookie.Value)
	originalSession, err := p.sessionService.GetByToken(reqCtx.Request.Context(), hashedOriginal)
	if err != nil || originalSession == nil {
		p.clearOriginalCookie(reqCtx.ResponseWriter, cookieName)
		return nil
	}

	// Verify the original session matches the impersonator's recorded session ID
	if sessionState.ImpersonatorSessionID == nil || *sessionState.ImpersonatorSessionID != originalSession.ID {
		p.clearOriginalCookie(reqCtx.ResponseWriter, cookieName)
		return nil
	}

	if originalSession.ExpiresAt.Before(time.Now().UTC()) {
		p.clearOriginalCookie(reqCtx.ResponseWriter, cookieName)
		return nil
	}

	if reqCtx.Actor.Claims == nil {
		reqCtx.Actor.Claims = make(map[string]any)
	}
	reqCtx.Actor.Claims[adminconstants.ImpersonatorID] = originalSession.UserID
	reqCtx.Actor.Claims[adminconstants.ImpersonatorOriginalSessionToken] = cookie.Value

	return nil
}

func (p *AdminPlugin) addImpersonationWhitelistScopes(reqCtx *models.RequestContext) error {
	if reqCtx.Actor == nil || reqCtx.Actor.Type != models.ActorUser {
		return nil
	}

	impersonatorID, hasID := reqCtx.Actor.GetClaimString(adminconstants.ImpersonatorID)
	if !hasID || impersonatorID == "" {
		return nil
	}

	whitelist := []string{adminconstants.SessionStateReadPermission, adminconstants.ImpersonationsStopPermission}

	seen := make(map[string]struct{}, len(reqCtx.Actor.Scopes)+len(whitelist))
	for _, s := range reqCtx.Actor.Scopes {
		seen[s] = struct{}{}
	}
	for _, s := range whitelist {
		if _, exists := seen[s]; !exists {
			reqCtx.Actor.Scopes = append(reqCtx.Actor.Scopes, s)
			seen[s] = struct{}{}
		}
	}

	return nil
}

func (p *AdminPlugin) clearOriginalCookie(w http.ResponseWriter, cookieName string) {
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName + adminconstants.OriginalSessionCookieSuffix,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}
