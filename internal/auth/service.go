package auth

import (
	"github.com/GoBetterAuth/go-better-auth/internal/auth/oauth2"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// Service encapsulates all authentication use-cases
type Service struct {
	config                 *domain.Config
	UserService            *UserService
	AccountService         *AccountService
	SessionService         *SessionService
	VerificationService    *VerificationService
	TokenService           *TokenService
	OAuth2ProviderRegistry *oauth2.OAuth2ProviderRegistry
}

// NewService creates a new Auth service with all dependencies
func NewService(
	config *domain.Config,
	userService *UserService,
	accountService *AccountService,
	sessionService *SessionService,
	verificationService *VerificationService,
	tokenService *TokenService,
) *Service {
	oauth2ProviderRegistry := oauth2.NewOAuth2ProviderRegistry()
	if config.SocialProviders.Default.Discord != nil {
		oauth2ProviderRegistry.Register(oauth2.NewDiscordProvider(config.SocialProviders.Default.Discord))
	}
	if config.SocialProviders.Default.GitHub != nil {
		oauth2ProviderRegistry.Register(oauth2.NewGitHubProvider(config.SocialProviders.Default.GitHub))
	}
	if config.SocialProviders.Default.Google != nil {
		oauth2ProviderRegistry.Register(oauth2.NewGoogleProvider(config.SocialProviders.Default.Google))
	}

	return &Service{
		config:                 config,
		UserService:            userService,
		AccountService:         accountService,
		SessionService:         sessionService,
		VerificationService:    verificationService,
		TokenService:           tokenService,
		OAuth2ProviderRegistry: oauth2ProviderRegistry,
	}
}
