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

	// HookIDSessionClear identifies the session clear hook.
	// Clears session cookie on sign-out
	HookIDSessionClear SessionHookID = "session.clear"
)

func (id SessionHookID) String() string {
	return string(id)
}

// validateSessionHook validates a session cookie from the request and sets UserID
// This hook runs at HookBefore stage if "session.auth" is in route.Metadata["plugins"]
func (p *SessionPlugin) validateSessionHook(reqCtx *models.RequestContext) error {
	p.logger.Debug("[validateSessionHook] checking method", "method", reqCtx.Method)

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

	// Optionally renew session if it's past 50% of its max age
	if p.shouldRenewSession(session) {
		p.renewSession(reqCtx.ResponseWriter, reqCtx.Request, session)
	}

	return nil
}

// issueSessionCookieHook hook handles generating sessions and setting the session cookie on successful authentication.
// This hook always runs by default at HookAfter stage.
// JWT tokens (access_token, refresh_token) are client-side only and sent in the response body,
// not as cookies. Only the session token is stored as a cookie for stateful session validation.
func (p *SessionPlugin) issueSessionCookieHook(reqCtx *models.RequestContext) error {
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
		// Session issuance hook: sets cookie after successful auth
		{
			Stage: models.HookAfter,
			Matcher: func(reqCtx *models.RequestContext) bool {
				sessionToken, ok := reqCtx.Values[models.ContextSessionToken.String()].(string)
				return ok && sessionToken != "" && reqCtx.UserID != nil
			},
			Handler: p.issueSessionCookieHook,
			Order:   5,
		},
		// Session clear hook: clears cookie on sign-out
		{
			Stage:    models.HookAfter,
			PluginID: HookIDSessionClear.String(),
			Handler:  p.clearSessionCookie,
			Order:    10,
		},
	}
}
