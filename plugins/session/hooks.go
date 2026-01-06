package session

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// Constants for session plugin hook IDs and metadata
const (
	// HookIDSessionAuth identifies the session authentication hook.
	// Validates session cookie and sets ctx.UserID if valid
	HookIDSessionAuth = "session.auth"

	// HookIDSessionIssuance identifies the session issuance hook.
	// Sets session cookie on successful authentication
	HookIDSessionIssuance = "session.issuance"

	// HookIDSessionContext overrides and returns a custom response with the session data.
	HookIDSessionContext = "session.context"

	// HookIDSessionClear identifies the session clear hook.
	// Clears session cookie on sign-out
	HookIDSessionClear = "session.clear"
)

// validateSessionHook validates a session cookie from the request and sets UserID
// This hook runs at HookBefore stage if "session.auth" is in route.Metadata["plugins"]
func (p *SessionPlugin) validateSessionHook(reqCtx *models.RequestContext) error {
	p.logger.Debug("session auth hook running", "path", reqCtx.Path, "method", reqCtx.Method)

	// Cooperative auth: if UserID already set by another auth plugin, skip
	if reqCtx.UserID != nil {
		p.logger.Debug("user already authenticated, skipping session validation")
		return nil
	}

	cookie, err := reqCtx.Request.Cookie(p.config.CookieName)
	if err != nil {
		p.logger.Debug("no session cookie found", "cookie_name", p.config.CookieName)
		reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		reqCtx.Handled = true
		return nil
	}

	sessionToken := cookie.Value
	p.logger.Debug("validating session token", "token", sessionToken)

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

// issueSessionCookieHook hook handles generating sessions and setting the session cookie on successful authentication
// This hook runs at HookAfter stage if "session.issuance" is in route.Metadata["plugins"]
// JWT tokens (access_token, refresh_token) are client-side only and sent in the response body,
// not as cookies. Only the session token is stored as a cookie for stateful session validation.
func (p *SessionPlugin) issueSessionCookieHook(reqCtx *models.RequestContext) error {
	if reqCtx.UserID == nil {
		return nil
	}

	authSuccess, ok := reqCtx.Values["auth_success"].(bool)
	if !ok || !authSuccess {
		return nil
	}

	if authSuccess {
		sessionToken, err := p.tokenService.Generate()
		if err != nil {
			return nil
		}

		hashedToken := p.tokenService.Hash(sessionToken)

		r := reqCtx.Request
		ctx := r.Context()

		ipAddress := util.ExtractClientIP(
			r.Header.Get("X-Forwarded-For"),
			r.Header.Get("X-Real-IP"),
			r.RemoteAddr,
		)
		userAgent := r.UserAgent()
		_, err = p.sessionService.Create(ctx, *reqCtx.UserID, hashedToken, &ipAddress, &userAgent, p.config.MaxAge)
		if err != nil {
			return nil
		}

		p.SetSessionCookie(reqCtx.ResponseWriter, sessionToken)
		return nil
	}

	return nil
}

func (p *SessionPlugin) sessionContextHook(ctx *models.RequestContext) error {
	if ctx.UserID == nil {
		return nil
	}

	session, err := p.sessionService.GetByUserID(ctx.Request.Context(), *ctx.UserID)
	if err != nil {
		p.logger.Error("failed to get session by user ID: %v", err)
		return nil
	}

	if ctx.ResponseBody != nil && ctx.ResponseReady {
		var payload map[string]any
		if err := json.Unmarshal(ctx.ResponseBody, &payload); err == nil {
			payload["session"] = session
			if updatedData, err := json.Marshal(payload); err == nil {
				ctx.ResponseBody = updatedData
			}
		}
	}

	return nil
}

// clearSessionCookie hook clears the session cookie on sign-out
// This hook runs at HookAfter stage if "session.clear" is in route.Metadata["plugins"]
// The application/router should only invoke this hook on sign-out routes via metadata
func (p *SessionPlugin) clearSessionCookie(ctx *models.RequestContext) error {
	p.logger.Debug("session clear hook running")
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
			PluginID: HookIDSessionAuth,
			Handler:  p.validateSessionHook,
			Order:    5,
		},

		// Session issuance hook: sets cookie after successful auth
		{
			Stage:    models.HookAfter,
			PluginID: HookIDSessionIssuance,
			Handler:  p.issueSessionCookieHook,
			Order:    5,
		},

		// Session context hook: returns session data in response
		{
			Stage:    models.HookAfter,
			PluginID: HookIDSessionContext,
			Handler:  p.sessionContextHook,
			Order:    10,
		},

		// Session clear hook: clears cookie on sign-out
		{
			Stage:    models.HookAfter,
			PluginID: HookIDSessionClear,
			Handler:  p.clearSessionCookie,
			Order:    10,
		},
	}
}
