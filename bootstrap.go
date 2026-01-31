package gobetterauth

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	"github.com/GoBetterAuth/go-better-auth/v2/events"
	internal "github.com/GoBetterAuth/go-better-auth/v2/internal"
	internalbootstrap "github.com/GoBetterAuth/go-better-auth/v2/internal/bootstrap"
	internalevents "github.com/GoBetterAuth/go-better-auth/v2/internal/events"
	internalrepositories "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	internalsecurity "github.com/GoBetterAuth/go-better-auth/v2/internal/security"
	internalservices "github.com/GoBetterAuth/go-better-auth/v2/internal/services"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	coreservices "github.com/GoBetterAuth/go-better-auth/v2/services"
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
	// Default to gochannel if not specified
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

	serviceRegistry.Register(models.ServiceUser.String(), userService)
	serviceRegistry.Register(models.ServiceAccount.String(), accountService)
	serviceRegistry.Register(models.ServiceSession.String(), sessionService)
	serviceRegistry.Register(models.ServiceVerification.String(), verificationService)
	serviceRegistry.Register(models.ServiceToken.String(), tokenService)

	return &coreservices.CoreServices{
		UserService:         userService,
		AccountService:      accountService,
		SessionService:      sessionService,
		VerificationService: verificationService,
		TokenService:        tokenService,
	}
}

func RunCoreMigrations(ctx context.Context, logger models.Logger, dbProvider string, db bun.IDB) error {
	sqlFS, err := internal.GetMigrations(ctx, dbProvider)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	if sqlFS == nil {
		logger.Debug("no migrations found for core")
	}

	migrations := migrate.NewMigrations()
	if err := migrations.Discover(*sqlFS); err != nil {
		logger.Debug("no migrations found for core")
		return nil
	}

	bunDB, ok := db.(*bun.DB)
	if !ok {
		logger.Debug("database is not *bun.DB, skipping migrations")
		return nil
	}

	m := migrate.NewMigrator(bunDB, migrations)

	if err := m.Init(ctx); err != nil {
		logger.Error("failed to init migrations table", "error", err)
		return err
	}

	if _, err := m.Migrate(ctx); err != nil {
		logger.Error("failed to run migrations", "error", err)
		return err
	}

	logger.Debug("core migrations completed")

	return nil
}

func DropCoreMigrations(ctx context.Context, logger models.Logger, dbProvider string, db bun.IDB) error {
	sqlFS, err := internal.GetMigrations(ctx, dbProvider)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	if sqlFS == nil {
		logger.Debug("no migrations found for core")
		return nil
	}

	migrations := migrate.NewMigrations()
	if err := migrations.Discover(*sqlFS); err != nil {
		logger.Debug("no migrations found for core")
		return nil
	}

	bunDB, ok := db.(*bun.DB)
	if !ok {
		logger.Debug("database is not *bun.DB, skipping drop migrations")
		return nil
	}

	m := migrate.NewMigrator(bunDB, migrations)

	if err := m.Init(ctx); err != nil {
		logger.Error("failed to init migrations table", "error", err)
		return err
	}

	// Rollback all applied migrations
	for {
		group, err := m.Rollback(ctx)
		if err != nil {
			logger.Error("failed to rollback migrations", "error", err)
			return err
		}
		if group == nil || len(group.Migrations) == 0 {
			break
		}
	}

	logger.Debug("core migrations dropped")

	return nil
}
