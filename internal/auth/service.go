package auth

import (
	"github.com/GoBetterAuth/go-better-auth/models"
	oauth2providers "github.com/GoBetterAuth/go-better-auth/oauth2-providers"
)

// Service encapsulates all services
type Service struct {
	config                 *models.Config
	EventBus               models.EventBus
	WebhookExecutor        models.WebhookExecutor
	EventEmitter           models.EventEmitter
	UserService            models.UserService
	AccountService         models.AccountService
	SessionService         models.SessionService
	VerificationService    models.VerificationService
	PasswordService        models.PasswordService
	TokenService           models.TokenService
	RateLimitService       models.RateLimitService
	MailerService          models.MailerService
	OAuth2ProviderRegistry *oauth2providers.OAuth2ProviderRegistry
}

// NewService creates a new Auth service with all dependencies
func NewService(
	config *models.Config,
	eventBus models.EventBus,
	webhookExecutor models.WebhookExecutor,
	eventEmitter models.EventEmitter,
	userService models.UserService,
	accountService models.AccountService,
	sessionService models.SessionService,
	verificationService models.VerificationService,
	passwordService models.PasswordService,
	tokenService models.TokenService,
	rateLimitService models.RateLimitService,
	mailerService models.MailerService,
	oauth2ProviderRegistry *oauth2providers.OAuth2ProviderRegistry,
) *Service {
	return &Service{
		config:                 config,
		EventBus:               eventBus,
		WebhookExecutor:        webhookExecutor,
		EventEmitter:           eventEmitter,
		UserService:            userService,
		AccountService:         accountService,
		SessionService:         sessionService,
		VerificationService:    verificationService,
		PasswordService:        passwordService,
		TokenService:           tokenService,
		RateLimitService:       rateLimitService,
		MailerService:          mailerService,
		OAuth2ProviderRegistry: oauth2ProviderRegistry,
	}
}
