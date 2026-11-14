package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/core/cleanup"
	"github.com/Authula/authula/env"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/repositories"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
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
	providerRegistry *services.ProviderRegistry
	httpClient       *http.Client
	useCases         *usecases.UseCases
	Api              *API

	oauthClientRepo       repositories.OAuthClientRepository
	authorizationCodeRepo repositories.AuthorizationCodeRepository
	consentRequestRepo    repositories.ConsentRequestRepository
	consentRepo           repositories.ConsentRepository
	loginTransactionRepo  repositories.LoginTransactionRepository
	flowStateRepo         repositories.FlowStateRepository

	stopCleanup chan struct{}
	done        chan struct{}
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
		Version:     "1.1.0",
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

	p.initializeRepositories()

	p.httpClient = &http.Client{Timeout: p.pluginConfig.HTTPTimeout}
	p.providerRegistry = services.NewProviderRegistry()
	if err := p.registerProviders(); err != nil {
		return fmt.Errorf("failed to register oauth providers: %w", err)
	}

	p.useCases = BuildUseCases(p)
	p.Api = &API{UseCases: p.useCases}

	if p.pluginConfig.AutoCleanup {
		p.stopCleanup = make(chan struct{})
		p.done = make(chan struct{})
		go p.runCleanupLoop(p.pluginConfig.CleanupInterval)
	}

	return nil
}

func (p *OAuth2Plugin) initializeRepositories() {
	p.oauthClientRepo = repositories.NewBunOAuthClientRepository(p.db)
	p.authorizationCodeRepo = repositories.NewBunAuthorizationCodeRepository(p.db)
	p.consentRequestRepo = repositories.NewBunConsentRequestRepository(p.db)
	p.consentRepo = repositories.NewBunConsentRepository(p.db)
	p.loginTransactionRepo = repositories.NewBunLoginTransactionRepository(p.db)
	p.flowStateRepo = repositories.NewBunFlowStateRepository(p.db)
}

func (p *OAuth2Plugin) Routes() []models.Route {
	return Routes(p)
}

func (p *OAuth2Plugin) Migrations(provider string) []migrations.Migration {
	return oauth2MigrationsForProvider(provider)
}

func (p *OAuth2Plugin) DependsOn() []string {
	return nil
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

	return nil
}

func getEnvKey(provider, suffix string) (string, bool) {
	switch provider {
	case "discord":
		switch suffix {
		case "CLIENT_ID":
			return env.EnvDiscordClientID, true
		case "CLIENT_SECRET":
			return env.EnvDiscordClientSecret, true
		}
	case "dropbox":
		switch suffix {
		case "CLIENT_ID":
			return env.EnvDropboxClientID, true
		case "CLIENT_SECRET":
			return env.EnvDropboxClientSecret, true
		}
	case "github":
		switch suffix {
		case "CLIENT_ID":
			return env.EnvGithubClientID, true
		case "CLIENT_SECRET":
			return env.EnvGithubClientSecret, true
		}
	case "google":
		switch suffix {
		case "CLIENT_ID":
			return env.EnvGoogleClientID, true
		case "CLIENT_SECRET":
			return env.EnvGoogleClientSecret, true
		}
	case "microsoft":
		switch suffix {
		case "CLIENT_ID":
			return env.EnvMicrosoftClientID, true
		case "CLIENT_SECRET":
			return env.EnvMicrosoftClientSecret, true
		}
	case "notion":
		switch suffix {
		case "CLIENT_ID":
			return env.EnvNotionClientID, true
		case "CLIENT_SECRET":
			return env.EnvNotionClientSecret, true
		}
	}
	return "", false
}

func (p *OAuth2Plugin) registerProviders() error {
	configured := make(map[string]types.ProviderConfig, len(p.pluginConfig.Providers))
	for name, cfg := range p.pluginConfig.Providers {
		canonical := strings.ToLower(strings.TrimSpace(name))
		if err := services.ValidateProviderName(canonical); err != nil {
			return fmt.Errorf("%w: %v", constants.ErrProviderConfigInvalid, err)
		}
		if _, exists := configured[canonical]; exists {
			return fmt.Errorf("%w: duplicate provider %s", constants.ErrProviderConfigInvalid, canonical)
		}
		configured[canonical] = cfg
	}

	names := make([]string, 0, len(configured))
	for name := range configured {
		names = append(names, name)
	}
	for _, builtIn := range []string{"discord", "dropbox", "github", "google", "microsoft", "notion"} {
		if _, exists := configured[builtIn]; !exists {
			names = append(names, builtIn)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		cfg, configuredProvider := configured[name]

		if configuredProvider && !cfg.IsEnabled() {
			continue
		}

		clientIDKey, ok := getEnvKey(name, "CLIENT_ID")
		if !ok {
			// Custom provider: every endpoint must be configured explicitly.
			if !configuredProvider {
				continue
			}
			if err := p.validateProviderConfig(name, cfg); err != nil {
				return err
			}
			if cfg.AuthURL == "" || cfg.TokenURL == "" || cfg.UserInfoURL == "" {
				return fmt.Errorf("%w: provider %s requires auth_url, token_url and user_info_url", constants.ErrProviderConfigInvalid, name)
			}
			if err := p.registerGenericProvider(name, cfg); err != nil {
				return err
			}
			continue
		}

		clientSecretKey, _ := getEnvKey(name, "CLIENT_SECRET")
		envClientID := strings.TrimSpace(os.Getenv(clientIDKey))
		envClientSecret := strings.TrimSpace(os.Getenv(clientSecretKey))
		if (envClientID == "") != (envClientSecret == "") {
			return fmt.Errorf("%w: provider %s", constants.ErrProviderConfigurationPartial, name)
		}

		if envClientID != "" {
			cfg.ClientID = envClientID
			cfg.ClientSecret = envClientSecret
			configuredProvider = true
		}
		if !configuredProvider {
			continue
		}

		if cfg.RedirectURL == "" {
			redirectURL, err := p.defaultRedirectURL(name)
			if err != nil {
				return err
			}
			cfg.RedirectURL = redirectURL
		}

		if err := p.validateProviderConfig(name, cfg); err != nil {
			return err
		}
		if err := p.registerBuiltInProvider(name, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (p *OAuth2Plugin) registerBuiltInProvider(name string, cfg types.ProviderConfig) error {
	var provider types.OAuth2Provider
	options := &services.ProviderOptions{HTTPClient: p.httpClient, Logger: p.logger}

	switch name {
	case "discord":
		provider = services.NewDiscordOAuth2Provider(cfg, options)
	case "dropbox":
		provider = services.NewDropboxOAuth2Provider(cfg, options)
	case "github":
		provider = services.NewGitHubOAuth2Provider(cfg, options)
	case "google":
		provider = services.NewGoogleOAuth2Provider(cfg, options)
	case "microsoft":
		if tenantID := strings.TrimSpace(os.Getenv(env.EnvMicrosoftTenantID)); tenantID != "" {
			cfg.TenantID = new(tenantID)
		}
		provider = services.NewMicrosoftOAuth2Provider(cfg, options)
	case "notion":
		provider = services.NewNotionOAuth2Provider(cfg, options)
	default:
		return fmt.Errorf("%w: unsupported built-in provider %s", constants.ErrProviderConfigInvalid, name)
	}

	if err := p.providerRegistry.Register(name, provider); err != nil {
		return fmt.Errorf("failed to register provider %s: %w", name, err)
	}
	return nil
}

func (p *OAuth2Plugin) registerGenericProvider(name string, cfg types.ProviderConfig) error {
	provider := services.NewGenericOAuth2Provider(name, cfg, &services.ProviderOptions{
		HTTPClient: p.httpClient,
		Logger:     p.logger,
	})
	if err := p.providerRegistry.Register(name, provider); err != nil {
		return fmt.Errorf("failed to register provider %s: %w", name, err)
	}
	return nil
}

func (p *OAuth2Plugin) defaultRedirectURL(provider string) (string, error) {
	base, err := url.Parse(p.globalConfig.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("%w: base_url must be an absolute URL", constants.ErrProviderConfigInvalid)
	}

	base.Path = path.Join(base.Path, p.globalConfig.BasePath, "/oauth2/callback/", provider)
	base.RawQuery = ""
	base.Fragment = ""

	redirectURL := base.String()
	if err := util.ValidateRedirectURL(redirectURL); err != nil {
		return "", fmt.Errorf("%w: provider %s %v", constants.ErrProviderConfigInvalid, provider, err)
	}
	return redirectURL, nil
}

func (p *OAuth2Plugin) validateProviderConfig(name string, cfg types.ProviderConfig) error {
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" || strings.TrimSpace(cfg.RedirectURL) == "" {
		return fmt.Errorf("%w: provider %s is missing client_id, client_secret or redirect_url", constants.ErrProviderConfigInvalid, name)
	}
	if err := util.ValidateRedirectURL(cfg.RedirectURL); err != nil {
		return fmt.Errorf("%w: provider %s: %v", constants.ErrProviderConfigInvalid, name, err)
	}

	for label, endpoint := range map[string]string{
		"auth_url":      cfg.AuthURL,
		"token_url":     cfg.TokenURL,
		"user_info_url": cfg.UserInfoURL,
	} {
		if endpoint == "" {
			continue
		}
		if err := validateEndpointURL(label, endpoint); err != nil {
			return fmt.Errorf("%w: provider %s %s", constants.ErrProviderConfigInvalid, name, err)
		}
	}

	return nil
}

func validateEndpointURL(label, endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", label)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", label)
	}
	if parsed.Scheme == "http" && !util.IsLocalhost(parsed.Host) {
		return fmt.Errorf("%s must use https except for localhost", label)
	}
	return nil
}

func (p *OAuth2Plugin) Close() error {
	if p.stopCleanup != nil {
		close(p.stopCleanup)
		<-p.done
		p.stopCleanup = nil
		p.done = nil
	}
	return nil
}

func (p *OAuth2Plugin) runCleanupLoop(interval time.Duration) {
	cleanup.RunLoop(p.stopCleanup, p.done, interval, func(ctx context.Context) {
		now := time.Now().UTC()
		if _, err := p.authorizationCodeRepo.DeleteExpired(ctx, now); err != nil {
			p.logger.Error("oauth2 authorization code cleanup failed", "error", err)
		}
		if _, err := p.consentRequestRepo.DeleteExpired(ctx, now); err != nil {
			p.logger.Error("oauth2 consent request cleanup failed", "error", err)
		}
		if _, err := p.loginTransactionRepo.DeleteExpired(ctx, now); err != nil {
			p.logger.Error("oauth2 login transaction cleanup failed", "error", err)
		}
		if _, err := p.flowStateRepo.DeleteExpired(ctx, now); err != nil {
			p.logger.Error("oauth2 flow state cleanup failed", "error", err)
		}
	})
}
