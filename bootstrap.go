package authula

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/events"
	internalbootstrap "github.com/Authula/authula/internal/bootstrap"
	internalevents "github.com/Authula/authula/internal/events"
	internalrepositories "github.com/Authula/authula/internal/repositories"
	internalsecurity "github.com/Authula/authula/internal/security"
	internalservices "github.com/Authula/authula/internal/services"
	internalsystemssession "github.com/Authula/authula/internal/systems/session"
	internalsystemsverification "github.com/Authula/authula/internal/systems/verification"
	"github.com/Authula/authula/models"
	coreservices "github.com/Authula/authula/services"
)

// InitLogger initializes the logger based on configuration
func InitLogger(config *models.Config) models.Logger {
	return internalbootstrap.InitLogger(internalbootstrap.LoggerOptions{
		Level:  config.Logger.Level,
		Logger: config.Logger.Logger,
	})
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

	pubsub, err := internalevents.InitWatermillProvider(&eventBusConfig, logger)
	if err != nil {
		return nil, err
	}

	return internalevents.NewEventBus(config, logger, pubsub), nil
}

func InitCoreServices(config *models.Config, db bun.IDB, serviceRegistry models.ServiceRegistry) *coreservices.CoreServices {
	signer := internalsecurity.NewHMACSigner(config.Secret)

	userRepo := internalrepositories.NewBunUserRepository(db)
	accountRepo := internalrepositories.NewBunAccountRepository(db)
	sessionRepo := internalrepositories.NewBunSessionRepository(db)
	verificationRepo := internalrepositories.NewBunVerificationRepository(db)
	tokenRepo := internalrepositories.NewCryptoTokenRepository(config.Secret)

	userService := internalservices.NewUserService(userRepo, config.CoreDatabaseHooks)
	accountService := internalservices.NewAccountService(config, accountRepo, tokenRepo, config.CoreDatabaseHooks)
	sessionService := internalservices.NewSessionService(sessionRepo, signer, config.CoreDatabaseHooks)
	verificationService := internalservices.NewVerificationService(verificationRepo, signer, config.CoreDatabaseHooks)
	tokenService := internalservices.NewTokenService(tokenRepo)
	passwordService := internalservices.NewArgon2PasswordService()

	serviceRegistry.Register(models.ServiceUser.String(), userService)
	serviceRegistry.Register(models.ServiceAccount.String(), accountService)
	serviceRegistry.Register(models.ServiceSession.String(), sessionService)
	serviceRegistry.Register(models.ServiceVerification.String(), verificationService)
	serviceRegistry.Register(models.ServiceToken.String(), tokenService)
	serviceRegistry.Register(models.ServicePassword.String(), passwordService)

	return &coreservices.CoreServices{
		UserService:         userService,
		AccountService:      accountService,
		SessionService:      sessionService,
		VerificationService: verificationService,
		TokenService:        tokenService,
		PasswordService:     passwordService,
	}
}

func InitCoreSystems(logger models.Logger, config *models.Config, coreServices *coreservices.CoreServices) []models.CoreSystem {
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
