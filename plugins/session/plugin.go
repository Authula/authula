package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type SessionPlugin struct {
	globalConfig   *models.Config
	pluginConfig   SessionPluginConfig
	ctx            *models.PluginContext
	logger         models.Logger
	userService    services.UserService
	sessionService services.SessionService
	tokenService   services.TokenService
}

func New(config SessionPluginConfig) *SessionPlugin {
	config.ApplyDefaults()
	return &SessionPlugin{pluginConfig: config}
}

func (p *SessionPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginSession.String(),
		Version:     "1.0.0",
		Description: "Provides cookie-based session authentication",
	}
}

func (p *SessionPlugin) Config() any {
	return p.pluginConfig
}

func (p *SessionPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger
	globalConfig := ctx.GetConfig()
	p.globalConfig = globalConfig

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.pluginConfig); err != nil {
		return err
	}

	p.pluginConfig.ApplyDefaults()

	userService, ok := ctx.ServiceRegistry.Get(models.ServiceUser.String()).(services.UserService)
	if !ok {
		p.logger.Error("user service not found in service registry")
		return errors.New("user service not available")
	}
	p.userService = userService

	sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(services.SessionService)
	if !ok {
		p.logger.Error("session service not found in service registry")
		return errors.New("session service not available")
	}
	p.sessionService = sessionService

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(services.TokenService)
	if !ok {
		p.logger.Error("token service not found in service registry")
		return errors.New("token service not available")
	}
	p.tokenService = tokenService

	if globalConfig.Session.UpdateAge >= globalConfig.Session.ExpiresIn {
		p.logger.Warn("session update_age is not shorter than expires_in; every authenticated request will extend the session",
			"update_age", globalConfig.Session.UpdateAge,
			"expires_in", globalConfig.Session.ExpiresIn,
		)
	}

	return nil
}

func (p *SessionPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *SessionPlugin) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, rawToken, err := p.validateSessionCookie(r)
			if err != nil {
				errorMsg := "unauthorized"
				statusCode := http.StatusUnauthorized
				p.writeErrorResponse(w, statusCode, errorMsg)
				return
			}

			if p.shouldRenewSession(session) {
				p.renewSession(w, r, session, rawToken)
			}

			ctx := context.WithValue(r.Context(), models.ContextUserID, session.UserID)
			ctx = context.WithValue(ctx, models.ContextSessionID, session.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (p *SessionPlugin) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if session, rawToken, err := p.validateSessionCookie(r); err == nil && session != nil {
				if p.shouldRenewSession(session) {
					p.renewSession(w, r, session, rawToken)
				}

				ctx := context.WithValue(r.Context(), models.ContextUserID, session.UserID)
				ctx = context.WithValue(ctx, models.ContextSessionID, session.ID)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// validateSessionCookie resolves the session behind the request's cookie and returns it
// together with the raw cookie value so the cookie can be re-issued unchanged on renewal.
func (p *SessionPlugin) validateSessionCookie(r *http.Request) (*models.Session, string, error) {
	cookie, err := r.Cookie(p.globalConfig.Session.CookieName)
	if err != nil {
		return nil, "", err
	}

	session, err := p.sessionService.GetByToken(r.Context(), p.tokenService.Hash(cookie.Value))
	if err != nil {
		return nil, "", err
	}
	if session == nil {
		return nil, "", fmt.Errorf("session not found")
	}

	if session.ExpiresAt.Before(time.Now().UTC()) {
		if err := p.sessionService.Delete(r.Context(), session.ID); err != nil {
			p.logger.Error("failed to delete expired session", "error", err)
		}
		return nil, "", fmt.Errorf("session expired")
	}

	return session, cookie.Value, nil
}

func (p *SessionPlugin) writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	util.JSONResponse(w, statusCode, map[string]string{
		"message": errorMsg,
	})
}

func (p *SessionPlugin) SetSessionCookie(w http.ResponseWriter, sessionToken string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     p.globalConfig.Session.CookieName,
		Value:    sessionToken,
		Path:     "/",
		Domain:   p.globalConfig.Session.Domain,
		HttpOnly: p.globalConfig.Session.HttpOnly,
		Secure:   p.globalConfig.Session.Secure,
		SameSite: util.ParseSameSite(p.globalConfig.Session.SameSite),
		MaxAge:   p.cookieMaxAge(expiresAt),
	})
}

func (p *SessionPlugin) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     p.globalConfig.Session.CookieName,
		Value:    "",
		Path:     "/",
		Domain:   p.globalConfig.Session.Domain,
		HttpOnly: p.globalConfig.Session.HttpOnly,
		Secure:   p.globalConfig.Session.Secure,
		SameSite: util.ParseSameSite(p.globalConfig.Session.SameSite),
		MaxAge:   -1,
	})
}

func (p *SessionPlugin) cookieMaxAge(expiresAt time.Time) int {
	remaining := time.Until(expiresAt)
	maxAge := p.globalConfig.Session.CookieMaxAge
	if maxAge <= 0 || remaining < maxAge {
		maxAge = remaining
	}
	return int(maxAge.Seconds())
}

func (p *SessionPlugin) shouldRenewSession(session *models.Session) bool {
	now := time.Now().UTC()
	timeToExpiry := session.ExpiresAt.Sub(now)
	return timeToExpiry <= p.globalConfig.Session.UpdateAge
}

// renewSession slides the session's expiry forward in place and re-issues the same cookie
// with a refreshed Max-Age. The row and token are kept, so concurrent requests carrying the
// same cookie all keep validating while the renewal is in flight.
func (p *SessionPlugin) renewSession(w http.ResponseWriter, r *http.Request, session *models.Session, rawToken string) {
	renewed := *session
	renewed.ExpiresAt = time.Now().UTC().Add(p.globalConfig.Session.ExpiresIn)

	updated, err := p.sessionService.Update(r.Context(), &renewed)
	if err != nil {
		p.logger.Error("session renewal failed: update error", "error", err)
		return
	}

	session.ExpiresAt = updated.ExpiresAt
	p.SetSessionCookie(w, rawToken, session.ExpiresAt)
}

func (p *SessionPlugin) Close() error {
	return nil
}
