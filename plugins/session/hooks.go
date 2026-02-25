package session

import (
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type SessionHookID string

const (
	HookIDSessionAuth         SessionHookID = "session.auth"
	HookIDSessionAuthOptional SessionHookID = "session.auth.optional"
)

func (id SessionHookID) String() string {
	return string(id)
}

func (p *SessionPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		{
			Stage:    models.HookBefore,
			PluginID: HookIDSessionAuth.String(),
			Handler:  p.validateSessionHook,
			Order:    10,
		},
		{
			Stage:    models.HookBefore,
			PluginID: HookIDSessionAuthOptional.String(),
			Handler:  p.validateSessionHookOptional,
			Order:    10,
		},
		{
			Stage:   models.HookAfter,
			Matcher: p.authSuccessMatcher,
			Handler: p.issueSessionCookieHook,
			Order:   10,
		},
		{
			Stage:   models.HookAfter,
			Matcher: p.signedOutMatcher,
			Handler: p.clearSessionCookieHook,
			Order:   10,
		},
	}
}

func (p *SessionPlugin) authSuccessMatcher(reqCtx *models.RequestContext) bool {
	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	return ok && authSuccess
}

func (p *SessionPlugin) signedOutMatcher(reqCtx *models.RequestContext) bool {
	signedOut, ok := reqCtx.Values[models.ContextAuthSignOut.String()].(bool)
	return ok && signedOut
}

func (p *SessionPlugin) validateSessionHook(reqCtx *models.RequestContext) error {
	// Cooperative auth: if UserID already set by another auth plugin, skip
	if reqCtx.UserID != nil {
		return nil
	}

	cookie, err := reqCtx.Request.Cookie(p.globalConfig.Session.CookieName)
	if err != nil {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	sessionToken := cookie.Value

	hashedToken := p.tokenService.Hash(sessionToken)
	session, err := p.sessionService.GetByToken(reqCtx.Request.Context(), hashedToken)
	if err != nil || session == nil {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	if session.ExpiresAt.Before(time.Now().UTC()) {
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	reqCtx.SetUserIDInContext(session.UserID)
	reqCtx.Values[models.ContextSessionID.String()] = session.ID

	// Optionally renew session if it's past 50% of its max age
	if p.shouldRenewSession(session) {
		p.renewSession(reqCtx.ResponseWriter, reqCtx.Request, session)
	}

	return nil
}

func (p *SessionPlugin) validateSessionHookOptional(reqCtx *models.RequestContext) error {
	// Cooperative auth: if UserID already set by another auth plugin, skip
	if reqCtx.UserID != nil {
		return nil
	}

	cookie, err := reqCtx.Request.Cookie(p.globalConfig.Session.CookieName)
	if err != nil {
		// No cookie, skip silently
		return nil
	}

	sessionToken := cookie.Value
	hashedToken := p.tokenService.Hash(sessionToken)
	session, err := p.sessionService.GetByToken(reqCtx.Request.Context(), hashedToken)
	if err != nil || session == nil {
		// Invalid session, skip silently
		return nil
	}

	if session.ExpiresAt.Before(time.Now().UTC()) {
		// Expired session, skip silently
		return nil
	}

	reqCtx.SetUserIDInContext(session.UserID)
	reqCtx.Values[models.ContextSessionID.String()] = session.ID

	// Optionally renew session if it's past 50% of its max age
	if p.shouldRenewSession(session) {
		p.renewSession(reqCtx.ResponseWriter, reqCtx.Request, session)
	}

	return nil
}

func (p *SessionPlugin) issueSessionCookieHook(reqCtx *models.RequestContext) error {
	if reqCtx.UserID == nil {
		return nil
	}

	sessionToken, ok := reqCtx.Values[models.ContextSessionToken.String()].(string)
	if !ok || sessionToken == "" {
		return nil
	}

	p.SetSessionCookie(reqCtx.ResponseWriter, sessionToken)

	return nil
}

func (p *SessionPlugin) clearSessionCookieHook(ctx *models.RequestContext) error {
	p.ClearSessionCookie(ctx.ResponseWriter)
	return nil
}
