package session

import (
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type SessionHookID string

// Constants for session plugin hook IDs and metadata
const (
	// HookIDSessionAuth identifies the session authentication hook.
	// Validates session cookie and sets ctx.UserID if valid
	HookIDSessionAuth SessionHookID = "session.auth"

	// HookIDSessionAuthOptional identifies the optional session authentication hook.
	// Validates session cookie and sets ctx.UserID if valid, but does not return unauthorized if invalid
	HookIDSessionAuthOptional SessionHookID = "session.auth.optional"
)

func (id SessionHookID) String() string {
	return string(id)
}

// authSuccessMatcher returns true if the request context indicates a successful authentication
func (p *SessionPlugin) authSuccessMatcher(reqCtx *models.RequestContext) bool {
	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	return ok && authSuccess
}

// signedOutMatcher returns true if the request context indicates a sign-out action
func (p *SessionPlugin) signedOutMatcher(reqCtx *models.RequestContext) bool {
	signedOut, ok := reqCtx.Values[models.ContextAuthSignOut.String()].(bool)
	return ok && signedOut
}

// validateSessionHook validates a session cookie from the request and sets UserID
// This hook runs at HookBefore stage if "session.auth" is in route.Metadata["plugins"]
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

// validateSessionHookOptional validates a session cookie from the request and sets UserID if valid.
// This hook runs at HookBefore stage if "session.auth.optional" is in route.Metadata["plugins"]
// Unlike validateSessionHook, it does not return unauthorized errors; it simply skips if no valid session.
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

// issueSessionCookieHook hook handles generating sessions and setting the session cookie on successful authentication.
// This hook always runs by default at HookAfter stage.
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

// clearSessionCookie hook clears the session cookie on sign-out
// This hook runs at HookAfter stage if "session.clear" is in route.Metadata["plugins"]
// The application/router should only invoke this hook on sign-out routes via metadata
func (p *SessionPlugin) clearSessionCookie(ctx *models.RequestContext) error {
	p.ClearSessionCookie(ctx.ResponseWriter)
	return nil
}

// buildHooks returns the configured hooks for this plugin
// Uses the new PluginID-based hook filtering for metadata-driven execution
func (p *SessionPlugin) buildHooks() []models.Hook {
	return []models.Hook{
		// Session authentication hook: validates cookie, sets UserID
		{
			Stage:    models.HookBefore,
			PluginID: HookIDSessionAuth.String(),
			Handler:  p.validateSessionHook,
			Order:    5,
		},
		// Optional session authentication hook: validates cookie, sets UserID if valid, no errors
		{
			Stage:    models.HookBefore,
			PluginID: HookIDSessionAuthOptional.String(),
			Handler:  p.validateSessionHookOptional,
			Order:    5,
		},
		// Session issuance hook: sets cookie after successful auth
		{
			Stage:   models.HookAfter,
			Matcher: p.authSuccessMatcher,
			Handler: p.issueSessionCookieHook,
			Order:   5,
		},
		// Session clear hook: clears cookie on sign-out
		{
			Stage:   models.HookAfter,
			Matcher: p.signedOutMatcher,
			Handler: p.clearSessionCookie,
			Order:   10,
		},
	}
}
