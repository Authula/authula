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
	config         SessionPluginConfig
	ctx            *models.PluginContext
	logger         models.Logger
	userService    services.UserService
	sessionService services.SessionService
	tokenService   services.TokenService
	Api            SessionAPI
}

func New(config SessionPluginConfig) *SessionPlugin {
	config.ApplyDefaults()
	return &SessionPlugin{config: config}
}

func (p *SessionPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginSession.String(),
		Version:     "1.0.0",
		Description: "Provides cookie-based session authentication",
	}
}

func (p *SessionPlugin) Config() any {
	return p.config
}

func (p *SessionPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	p.config.ApplyDefaults()

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

	p.Api = newSessionAPI(p)

	return nil
}

func (p *SessionPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *SessionPlugin) OnConfigUpdate(config *models.Config) error {
	if err := util.LoadPluginConfig(config, p.Metadata().ID, &p.config); err != nil {
		p.logger.Error("failed to parse session plugin config on update", "error", err)
		return err
	}

	p.config.ApplyDefaults()

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
				if err.Error() == "session_expired" {
					errorMsg = "session_expired"
					p.ClearSessionCookie(w)
				}
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
	cookie, err := r.Cookie(p.config.CookieName)
	if err != nil {
		return nil, err
	}

	session, err := p.sessionService.GetByToken(r.Context(), p.tokenService.Hash(cookie.Value))
	if err != nil || session == nil {
		return nil, err
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		_ = p.sessionService.Delete(r.Context(), session.ID)
		return nil, fmt.Errorf("session_expired")
	}

	return session, nil
}

func (p *SessionPlugin) writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	util.JSONResponse(w, statusCode, map[string]string{
		"message": errorMsg,
	})
}

func (p *SessionPlugin) getSameSiteMode() http.SameSite {
	switch p.config.SameSite {
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
		Name:     p.config.CookieName,
		Value:    sessionToken,
		Path:     p.config.CookiePath,
		HttpOnly: p.config.HttpOnly,
		Secure:   p.config.Secure,
		SameSite: sameSite,
		MaxAge:   int(p.config.MaxAge.Seconds()),
	})
}

func (p *SessionPlugin) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     p.config.CookieName,
		Value:    "",
		Path:     p.config.CookiePath,
		HttpOnly: p.config.HttpOnly,
		Secure:   p.config.Secure,
		MaxAge:   -1,
	})
}

// shouldRenewSession checks if the session is past 50% of its max age and should be renewed
func (p *SessionPlugin) shouldRenewSession(session *models.Session) bool {
	if session == nil {
		return false
	}
	now := time.Now().UTC()
	timeSinceCreation := now.Sub(session.CreatedAt)
	totalLifetime := session.ExpiresAt.Sub(session.CreatedAt)

	// Renew if more than 50% of the session lifetime has passed
	return timeSinceCreation > totalLifetime/2
}

// renewSession extends the session expiration in the database and updates the cookie
func (p *SessionPlugin) renewSession(w http.ResponseWriter, r *http.Request, session *models.Session) {
	newExpiresAt := time.Now().UTC().Add(p.config.MaxAge)
	if _, err := p.ctx.DB.NewUpdate().Model(session).Set("expires_at = ?", newExpiresAt).Where("id = ?", session.ID).Exec(r.Context()); err != nil {
		p.logger.Error("failed to renew session", "error", err, "session_id", session.ID)
		return
	}

	cookie, err := r.Cookie(p.config.CookieName)
	if err == nil && cookie != nil {
		p.SetSessionCookie(w, cookie.Value)
		p.logger.Debug("session renewed", "session_id", session.ID)
	}
}

func (p *SessionPlugin) Close() error {
	return nil
}
