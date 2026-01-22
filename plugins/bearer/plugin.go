package bearer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
)

// BearerPlugin validates Authorization headers using the JWT plugin.
type BearerPlugin struct {
	config     BearerPluginConfig
	logger     models.Logger
	ctx        *models.PluginContext
	jwtService services.JWTService
}

// New creates a new BearerPlugin instance with sane defaults.
func New(config BearerPluginConfig) *BearerPlugin {
	config.ApplyDefaults()
	return &BearerPlugin{config: config}
}

func (p *BearerPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginBearer.String(),
		Version:     "1.0.0",
		Description: "Validates Authorization headers using JWT access tokens",
	}
}

func (p *BearerPlugin) Config() any {
	return p.config
}

func (p *BearerPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}
	p.config.ApplyDefaults()

	jwtService, ok := ctx.ServiceRegistry.Get(models.ServiceJWT.String()).(services.JWTService)
	if !ok {
		return errors.New("jwt service is required for bearer plugin")
	}

	p.jwtService = jwtService

	return nil
}

func (p *BearerPlugin) Close() error {
	return nil
}

func (p *BearerPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *BearerPlugin) OnConfigUpdate(config *models.Config) error {
	if err := util.LoadPluginConfig(config, p.Metadata().ID, &p.config); err != nil {
		p.logger.Error("failed to parse bearer plugin config on update", "error", err)
		return err
	}

	p.config.ApplyDefaults()

	return nil
}

func (p *BearerPlugin) AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := p.extractToken(r)
			if err != nil {
				p.writeUnauthorized(w, err)
				return
			}

			userID, err := p.jwtService.ValidateToken(token)
			if err != nil {
				p.writeUnauthorized(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), models.ContextUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (p *BearerPlugin) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := p.extractToken(r)
			if err == nil && token != "" {
				if userID, validateErr := p.jwtService.ValidateToken(token); validateErr == nil {
					ctx := context.WithValue(r.Context(), models.ContextUserID, userID)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (p *BearerPlugin) extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get(p.config.HeaderName)
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	// Parse "Bearer <token>" format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization format: expected 'Bearer <token>'")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("empty bearer token")
	}

	return token, nil
}

func (p *BearerPlugin) writeUnauthorized(w http.ResponseWriter, err error) {
	util.JSONResponse(w, http.StatusUnauthorized, map[string]string{
		"message": err.Error(),
	})
}
