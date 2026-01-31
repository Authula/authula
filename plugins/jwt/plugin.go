package jwt

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/repositories"
	jwtservices "github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type JWTPlugin struct {
	globalConfig     *models.Config
	pluginConfig     types.JWTPluginConfig
	ctx              *models.PluginContext
	Logger           models.Logger
	sessionService   services.SessionService
	tokenService     services.TokenService
	jwtService       *jwtservices.JWTServiceImpl
	refreshService   jwtservices.RefreshTokenService
	keyService       jwtservices.KeyService
	cacheService     jwtservices.CacheService
	secondaryStorage models.SecondaryStorage
	blacklistService jwtservices.BlacklistService
}

func New(config types.JWTPluginConfig) *JWTPlugin {
	config.ApplyDefaults()
	return &JWTPlugin{pluginConfig: config}
}

func (p *JWTPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginJWT.String(),
		Version:     "1.0.0",
		Description: "JWKS-based JWT authentication with Ed25519 support among other algorithms",
	}
}

func (p *JWTPlugin) Config() any {
	return p.pluginConfig
}

func (p *JWTPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.Logger = ctx.Logger
	p.globalConfig = ctx.GetConfig()

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.pluginConfig); err != nil {
		return err
	}

	if err := p.pluginConfig.NormalizeAlgorithm(); err != nil {
		p.Logger.Error("invalid jwt algorithm in plugin config", "error", err)
		return err
	}

	sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(services.SessionService)
	if !ok {
		p.Logger.Error("session service not found")
		return errors.New("session service not available")
	}
	p.sessionService = sessionService

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(services.TokenService)
	if !ok {
		p.Logger.Error("token service not found")
		return errors.New("token service not available")
	}
	p.tokenService = tokenService

	if ss, ok := ctx.ServiceRegistry.Get(models.ServiceSecondaryStorage.String()).(services.SecondaryStorageService); ok {
		p.secondaryStorage = ss.GetStorage()
	}

	jwksRepo := repositories.NewBunJWKSRepository(ctx.DB)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(ctx.DB)

	p.keyService = jwtservices.NewKeyService(jwksRepo, p.Logger, p.tokenService, p.globalConfig.Secret, p.pluginConfig.Algorithm)
	p.cacheService = jwtservices.NewCacheService(jwksRepo, p.secondaryStorage, p.Logger, p.pluginConfig.JWKSCacheTTL)

	if p.secondaryStorage == nil {
		p.Logger.Warn("secondary storage not available; token blacklisting will be disabled")
	} else {
		p.blacklistService = jwtservices.NewBlacklistService(p.secondaryStorage, p.Logger)
	}

	if err := p.keyService.GenerateKeysIfMissing(context.Background()); err != nil {
		p.Logger.Error("failed to generate keys", "error", err)
		return fmt.Errorf("failed to generate keys: %w", err)
	}

	rotated, err := p.keyService.RotateKeysIfNeeded(
		context.Background(),
		p.pluginConfig.KeyRotationInterval,
		p.pluginConfig.KeyRotationGracePeriod,
		func(ctx context.Context) error {
			return p.cacheService.InvalidateCache(ctx)
		},
	)
	if err != nil {
		p.Logger.Warn("failed to check/rotate keys on startup", "error", err)
	} else if rotated {
		p.Logger.Info("key rotation occurred on startup due to interval expiration")
	}

	if err := p.cacheService.InvalidateCache(context.Background()); err != nil {
		p.Logger.Warn("failed to pre-populate cache on startup", "error", err)
	}

	jwtServiceImpl, ok := jwtservices.NewJWTService(
		p.Logger,
		p.sessionService,
		p.tokenService,
		p.keyService,
		p.cacheService,
		p.blacklistService,
		p.pluginConfig.ExpiresIn,
		p.pluginConfig.RefreshExpiresIn,
	).(*jwtservices.JWTServiceImpl)
	if !ok {
		return errors.New("failed to create JWT service")
	}
	p.jwtService = jwtServiceImpl

	p.refreshService = jwtservices.NewRefreshTokenService(
		p.Logger,
		ctx.EventBus,
		p.sessionService,
		p.jwtService,
		refreshTokenRepo,
		p.pluginConfig.RefreshGracePeriod,
		p.pluginConfig.RefreshExpiresIn,
	)

	ctx.ServiceRegistry.Register(models.ServiceJWT.String(), jwtServiceImpl)

	return nil
}

func (p *JWTPlugin) Migrations(ctx context.Context, dbProvider string) (*embed.FS, error) {
	return GetMigrations(ctx, dbProvider)
}

func (p *JWTPlugin) Routes() []models.Route {
	return Routes(p)
}

func (p *JWTPlugin) Hooks() []models.Hook {
	return p.buildHooks()
}

func (p *JWTPlugin) OnConfigUpdate(config *models.Config) error {
	if pluginCfg, ok := config.Plugins[models.PluginJWT.String()]; ok {
		if err := util.ParsePluginConfig(pluginCfg, &p.pluginConfig); err != nil {
			p.Logger.Error("failed to parse jwt plugin config on update", "error", err)
			return err
		}
	}

	p.pluginConfig.ApplyDefaults()
	if err := p.pluginConfig.NormalizeAlgorithm(); err != nil {
		p.Logger.Error("invalid jwt algorithm in plugin config update", "error", err)
		return err
	}

	rotated, err := p.keyService.RotateKeysIfNeeded(
		context.Background(),
		p.pluginConfig.KeyRotationInterval,
		p.pluginConfig.KeyRotationGracePeriod,
		func(ctx context.Context) error {
			return p.cacheService.InvalidateCache(ctx)
		},
	)
	if err != nil {
		p.Logger.Warn("failed to rotate keys after config update", "error", err)
	} else if rotated {
		p.Logger.Info("key rotation occurred after config update")
	}

	if p.cacheService != nil {
		if err := p.cacheService.InvalidateCache(context.Background()); err != nil {
			p.Logger.Error("failed to invalidate JWKS cache on config update", "error", err)
		}
	}

	return nil
}

func (p *JWTPlugin) Close() error {
	return nil
}
