package authula

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/uptrace/bun"

	coreevents "github.com/Authula/authula/core/events"
	corerepositories "github.com/Authula/authula/core/repositories"
	coreservices "github.com/Authula/authula/core/services"
	"github.com/Authula/authula/events"
	internalbootstrap "github.com/Authula/authula/internal/bootstrap"
	internalsecurity "github.com/Authula/authula/internal/security"
	internalsystemssession "github.com/Authula/authula/internal/systems/session"
	internalsystemsverification "github.com/Authula/authula/internal/systems/verification"
	"github.com/Authula/authula/models"
	serviceinterfaces "github.com/Authula/authula/services"
)

// InitLogger initializes the logger based on configuration
func InitLogger(config *models.Config) models.Logger {
	return internalbootstrap.InitLogger(internalbootstrap.LoggerOptions{Level: config.Logger.Level})
}

// InitDatabase creates a Bun DB connection based on provider
func InitDatabase(config *models.Config, logger models.Logger, logLevel string) (bun.IDB, error) {
	return internalbootstrap.InitDatabase(
		internalbootstrap.DatabaseOptions{
			Provider:        config.Database.Provider,
			URL:             config.Database.URL,
			MaxOpenConns:    config.Database.MaxOpenConns,
			MaxIdleConns:    config.Database.MaxIdleConns,
			ConnMaxLifetime: config.Database.ConnMaxLifetime,
		},
		logger,
		logLevel,
	)
}

// InitEventBus creates an event bus based on the configuration
func InitEventBus(config *models.Config) (models.EventBus, error) {
	provider := config.EventBus.Provider
	if provider == "" {
		provider = events.ProviderGoChannel
	}

	eventBusConfig := config.EventBus
	if provider == events.ProviderGoChannel && eventBusConfig.GoChannel == nil {
		eventBusConfig.GoChannel = &models.GoChannelConfig{
			BufferSize: 100,
		}
	}

	logger := watermill.NewStdLogger(false, false)

	pubsub, err := coreevents.InitWatermillProvider(&eventBusConfig, logger)
	if err != nil {
		return nil, err
	}

	return coreevents.NewEventBus(config, logger, pubsub), nil
}

func InitCoreServices(config *models.Config, db bun.IDB, serviceRegistry models.ServiceRegistry) *serviceinterfaces.CoreServices {
	signer := internalsecurity.NewHMACSigner(config.Secret)

	userRepo := corerepositories.NewBunUserRepository(db)
	accountRepo := corerepositories.NewBunAccountRepository(db)
	sessionRepo := corerepositories.NewBunSessionRepository(db)
	verificationRepo := corerepositories.NewBunVerificationRepository(db)
	tokenRepo := corerepositories.NewCryptoTokenRepository(config.Secret)

	userService := coreservices.NewUserService(userRepo, config.CoreDatabaseHooks)
	accountService := coreservices.NewAccountService(config, accountRepo, tokenRepo, config.CoreDatabaseHooks)
	sessionService := coreservices.NewSessionService(sessionRepo, signer, config.CoreDatabaseHooks)
	verificationService := coreservices.NewVerificationService(verificationRepo, signer, config.CoreDatabaseHooks)
	tokenService := coreservices.NewTokenService(tokenRepo)
	passwordService := coreservices.NewArgon2PasswordService()

	serviceRegistry.Register(models.ServiceUser.String(), userService)
	serviceRegistry.Register(models.ServiceAccount.String(), accountService)
	serviceRegistry.Register(models.ServiceSession.String(), sessionService)
	serviceRegistry.Register(models.ServiceVerification.String(), verificationService)
	serviceRegistry.Register(models.ServiceToken.String(), tokenService)
	serviceRegistry.Register(models.ServicePassword.String(), passwordService)

	return &serviceinterfaces.CoreServices{
		UserService:         userService,
		AccountService:      accountService,
		SessionService:      sessionService,
		VerificationService: verificationService,
		TokenService:        tokenService,
		PasswordService:     passwordService,
	}
}

func InitCoreSystems(logger models.Logger, config *models.Config, coreServices *serviceinterfaces.CoreServices) []models.CoreSystem {
	return []models.CoreSystem{
		internalsystemssession.NewSessionCleanupSystem(
			logger,
			config.Session,
			coreServices.SessionService,
		),
		internalsystemsverification.NewVerificationCleanupSystem(
			logger,
			config.Verification,
			coreServices.VerificationService,
		),
	}
}
