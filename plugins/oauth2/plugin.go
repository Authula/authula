package oauth2

import (
	"fmt"
	"os"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/env"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type OAuth2Plugin struct {
	globalConfig     *models.Config
	pluginConfig     types.OAuth2PluginConfig
	logger           models.Logger
	ctx              *models.PluginContext
	db               bun.IDB
	userService      rootservices.UserService
	accountService   rootservices.AccountService
	sessionService   rootservices.SessionService
	tokenService     rootservices.TokenService
	eventBus         models.EventBus
	providerRegistry *services.ProviderRegistry
	hmacKey          []byte
	Api              *API
}

func New(config types.OAuth2PluginConfig) *OAuth2Plugin {
	config.ApplyDefaults()
	return &OAuth2Plugin{
		pluginConfig: config,
	}
}

func (p *OAuth2Plugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginOAuth2.String(),
		Version:     "1.0.0",
		Description: "OAuth2 authentication plugin with support for various providers.",
	}
}

func (p *OAuth2Plugin) Config() any {
	return p.pluginConfig
}

func (p *OAuth2Plugin) Init(ctx *models.PluginContext) error {
	p.logger = ctx.Logger
	p.ctx = ctx
	p.db = ctx.DB
	p.globalConfig = ctx.GetConfig()

	if err := util.LoadPluginConfig(p.globalConfig, p.Metadata().ID, &p.pluginConfig); err != nil {
		return fmt.Errorf("failed to load oauth2 config: %w", err)
	}
	p.pluginConfig.ApplyDefaults()

	if err := p.initializeServices(ctx); err != nil {
		return err
	}

	p.providerRegistry = services.NewProviderRegistry()
	if err := p.registerBuiltInProviders(); err != nil {
		return fmt.Errorf("failed to register built-in providers: %w", err)
	}

	p.Api = BuildAPI(p)

	return nil
}

func (p *OAuth2Plugin) Routes() []models.Route {
	return Routes(p)
}

func (p *OAuth2Plugin) initializeServices(ctx *models.PluginContext) error {
	var ok bool

	p.userService, ok = ctx.ServiceRegistry.Get(models.ServiceUser.String()).(rootservices.UserService)
	if !ok {
		return fmt.Errorf("user service not available in service registry")
	}

	p.accountService, ok = ctx.ServiceRegistry.Get(models.ServiceAccount.String()).(rootservices.AccountService)
	if !ok {
		return fmt.Errorf("account service not available in service registry")
	}

	p.sessionService, ok = ctx.ServiceRegistry.Get(models.ServiceSession.String()).(rootservices.SessionService)
	if !ok {
		return fmt.Errorf("session service not available in service registry")
	}

	p.tokenService, ok = ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("token service not available in service registry")
	}

	p.eventBus = ctx.EventBus
	p.hmacKey = services.DeriveOAuthHMACKey(p.globalConfig.Secret)

	return nil
}

func getEnvKey(provider, suffix string) string {
	switch provider {
	case "discord":
		if suffix == "CLIENT_ID" {
			return env.EnvDiscordClientID
		}
		if suffix == "CLIENT_SECRET" {
			return env.EnvDiscordClientSecret
		}
	case "github":
		if suffix == "CLIENT_ID" {
			return env.EnvGithubClientID
		}
		if suffix == "CLIENT_SECRET" {
			return env.EnvGithubClientSecret
		}
	case "google":
		if suffix == "CLIENT_ID" {
			return env.EnvGoogleClientID
		}
		if suffix == "CLIENT_SECRET" {
			return env.EnvGoogleClientSecret
		}
	}
	return ""
}

func (p *OAuth2Plugin) registerBuiltInProviders() error {
	providers := map[string]struct {
		validator func(types.ProviderConfig) error
		creator   func(types.ProviderConfig) types.OAuth2Provider
	}{
		"discord": {
			validator: p.validateProviderConfig,
			creator: func(cfg types.ProviderConfig) types.OAuth2Provider {
				return services.NewDiscordProvider(cfg.ClientID, cfg.ClientSecret, cfg.RedirectURL)
			},
		},
		"github": {
			validator: p.validateProviderConfig,
			creator: func(cfg types.ProviderConfig) types.OAuth2Provider {
				return services.NewGitHubProvider(cfg.ClientID, cfg.ClientSecret, cfg.RedirectURL)
			},
		},
		"google": {
			validator: p.validateProviderConfig,
			creator: func(cfg types.ProviderConfig) types.OAuth2Provider {
				return services.NewGoogleProvider(cfg.ClientID, cfg.ClientSecret, cfg.RedirectURL)
			},
		},
	}

	for name, provider := range providers {
		var providerConfig types.ProviderConfig
		clientID := os.Getenv(getEnvKey(name, "CLIENT_ID"))
		clientSecret := os.Getenv(getEnvKey(name, "CLIENT_SECRET"))
		if clientID != "" && clientSecret != "" {
			providerConfig = types.ProviderConfig{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURL:  p.globalConfig.BaseURL + p.globalConfig.BasePath + "/oauth2/callback/" + name,
			}
		} else if config, ok := p.pluginConfig.Providers[name]; ok {
			providerConfig = config
		} else {
			continue // skip if neither env vars nor config is set
		}
		if err := provider.validator(providerConfig); err != nil {
			return err
		}
		p.providerRegistry.Register(name, provider.creator(providerConfig))
	}

	return nil
}

func (p *OAuth2Plugin) validateProviderConfig(config types.ProviderConfig) error {
	if config.ClientID == "" || config.ClientSecret == "" || config.RedirectURL == "" {
		return fmt.Errorf("provider config missing required fields (ClientID, ClientSecret, RedirectURL)")
	}
	return nil
}

func (p *OAuth2Plugin) Close() error {
	return nil
}
