package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
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

	return nil
}

func (p *SessionPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *SessionPlugin) OnConfigUpdate(config *models.Config) error {
	if err := util.LoadPluginConfig(config, p.Metadata().ID, &p.pluginConfig); err != nil {
		p.logger.Error("failed to parse session plugin config on update", "error", err)
		return err
	}

	p.pluginConfig.ApplyDefaults()

	return nil
}

// AuthMiddleware validates session cookie and extracts user ID
func (p *SessionPlugin) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := p.validateSessionCookie(r)
			if err != nil {
				errorMsg := "unauthorized"
				statusCode := http.StatusUnauthorized
				p.writeErrorResponse(w, statusCode, errorMsg)
				return
			}

			// Check if session should be renewed (sliding window: <50% life remaining)
			if p.shouldRenewSession(session) {
				p.renewSession(w, r, session)
			}

			ctx := context.WithValue(r.Context(), models.ContextUserID, session.UserID)
			ctx = context.WithValue(ctx, models.ContextSessionID, session.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware validates session if present but doesn't require it
func (p *SessionPlugin) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if session, err := p.validateSessionCookie(r); err == nil && session != nil {
				// Check if session should be renewed (sliding window: <50% life remaining)
				if p.shouldRenewSession(session) {
					p.renewSession(w, r, session)
				}

				ctx := context.WithValue(r.Context(), models.ContextUserID, session.UserID)
				ctx = context.WithValue(ctx, models.ContextSessionID, session.ID)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (p *SessionPlugin) validateSessionCookie(r *http.Request) (*models.Session, error) {
	cookie, err := r.Cookie(p.globalConfig.Session.CookieName)
	if err != nil {
		return nil, err
	}

	session, err := p.sessionService.GetByToken(r.Context(), p.tokenService.Hash(cookie.Value))
	if err != nil || session == nil {
		return nil, err
	}

	if session.ExpiresAt.Before(time.Now().UTC()) {
		p.sessionService.Delete(r.Context(), session.ID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func (p *SessionPlugin) writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	util.JSONResponse(w, statusCode, map[string]string{
		"message": errorMsg,
	})
}

func (p *SessionPlugin) getSameSiteMode() http.SameSite {
	switch p.globalConfig.Session.SameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}

func (p *SessionPlugin) SetSessionCookie(w http.ResponseWriter, sessionToken string) {
	sameSite := p.getSameSiteMode()

	http.SetCookie(w, &http.Cookie{
		Name:     p.globalConfig.Session.CookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: p.globalConfig.Session.HttpOnly,
		Secure:   p.globalConfig.Session.Secure,
		SameSite: sameSite,
		MaxAge:   int(p.globalConfig.Session.CookieMaxAge.Seconds()),
	})
}

func (p *SessionPlugin) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     p.globalConfig.Session.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: p.globalConfig.Session.HttpOnly,
		Secure:   p.globalConfig.Session.Secure,
		MaxAge:   -1,
	})
}

// shouldRenewSession checks if the session is past 50% of its max age and should be renewed
func (p *SessionPlugin) shouldRenewSession(session *models.Session) bool {
	now := time.Now().UTC()
	timeToExpiry := session.ExpiresAt.Sub(now)
	return timeToExpiry <= p.globalConfig.Session.UpdateAge
}

// renewSession extends the session expiration in the database and updates the cookie
func (p *SessionPlugin) renewSession(w http.ResponseWriter, r *http.Request, session *models.Session) {
	cookie, _ := r.Cookie(p.globalConfig.Session.CookieName)
	if cookie == nil {
		return
	}

	session.ExpiresAt = time.Now().UTC().Add(p.globalConfig.Session.ExpiresIn)
	if _, err := p.sessionService.Update(r.Context(), session); err != nil {
		p.logger.Error("session renewal failed", "error", err)
		return
	}

	p.SetSessionCookie(w, cookie.Value)
}

func (p *SessionPlugin) Close() error {
	return nil
}
