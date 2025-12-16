package auth

import (
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/providers"
)

// Service encapsulates all authentication use-cases
type Service struct {
	config                 *models.Config
	EventBus               models.EventBus
	UserService            models.UserService
	AccountService         models.AccountService
	SessionService         models.SessionService
	VerificationService    models.VerificationService
	TokenService           models.TokenService
	RateLimitService       models.RateLimitService
	OAuth2ProviderRegistry *providers.OAuth2ProviderRegistry
}

// NewService creates a new Auth service with all dependencies
func NewService(
	config *models.Config,
	eventBus models.EventBus,
	userService models.UserService,
	accountService models.AccountService,
	sessionService models.SessionService,
	verificationService models.VerificationService,
	tokenService models.TokenService,
	rateLimitService models.RateLimitService,
) *Service {
	oauth2ProviderRegistry := providers.NewOAuth2ProviderRegistry()
	if config.SocialProviders.Default.Discord != nil {
		oauth2ProviderRegistry.Register(providers.NewDiscordProvider(config.SocialProviders.Default.Discord))
	}
	if config.SocialProviders.Default.GitHub != nil {
		oauth2ProviderRegistry.Register(providers.NewGitHubProvider(config.SocialProviders.Default.GitHub))
	}
	if config.SocialProviders.Default.Google != nil {
		oauth2ProviderRegistry.Register(providers.NewGoogleProvider(config.SocialProviders.Default.Google))
	}

	return &Service{
		config:                 config,
		EventBus:               eventBus,
		UserService:            userService,
		AccountService:         accountService,
		SessionService:         sessionService,
		VerificationService:    verificationService,
		TokenService:           tokenService,
		RateLimitService:       rateLimitService,
		OAuth2ProviderRegistry: oauth2ProviderRegistry,
	}
}
