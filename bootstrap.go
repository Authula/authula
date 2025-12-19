package gobetterauth

import (
	"log/slog"
	"os"

	"fmt"

	"gorm.io/gorm"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"

	"github.com/GoBetterAuth/go-better-auth/events"
	internalauth "github.com/GoBetterAuth/go-better-auth/internal/auth"
	internalconfig "github.com/GoBetterAuth/go-better-auth/internal/config"
	internalevents "github.com/GoBetterAuth/go-better-auth/internal/events"
	"github.com/GoBetterAuth/go-better-auth/internal/plugins"
	"github.com/GoBetterAuth/go-better-auth/internal/services"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/providers"
	"github.com/GoBetterAuth/go-better-auth/storage"
)

// -------------------------------
// DEFAULTS
// -------------------------------

// initLogger initializes the logger based on configuration
func initLogger(config *models.Config) {
	if config.Logger.Logger != nil {
		return
	}

	var level slog.Level
	switch config.Logger.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	config.Logger.Logger = logger
}

func InitDefaults(config *models.Config) {
	util.InitValidator()
	initLogger(config)
}

// -------------------------------
// DATABASE
// -------------------------------

// InitDatabase creates a GORM DB connection based on provider.
func InitDatabase(config *models.Config) (*gorm.DB, error) {
	logger := config.Logger.Logger

	if config.Mode != models.ModeLibrary {
		// Validate configuration values at startup to catch errors early.
		if config.Database.Provider == "" && config.DB == nil {
			return nil, fmt.Errorf("database provider must be specified or DB must be pre-initialized")
		}
		if config.Database.ConnectionString == "" && config.DB == nil {
			return nil, fmt.Errorf("database connection string must be specified or DB must be pre-initialized")
		}
	}

	// Initialize DB using the configured DatabaseConfig if not already set.
	if config.DB == nil {
		var dialector gorm.Dialector
		switch config.Database.Provider {
		case "sqlite":
			dialector = sqlite.Open(config.Database.ConnectionString)
		case "postgres":
			dialector = postgres.Open(config.Database.ConnectionString)
		case "mysql":
			dialector = mysql.Open(config.Database.ConnectionString)
		default:
			return nil, fmt.Errorf("unsupported database provider: %s", config.Database.Provider)
		}

		// Create GORM config and suppress query logging unless in debug mode
		gormConfig := &gorm.Config{
			SkipDefaultTransaction: true,
		}

		// In non-debug modes, disable GORM's internal logger
		if config.Logger.Level != "debug" {
			gormConfig.Logger = nil
		}

		db, err := gorm.Open(dialector, gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to open database connection: %w", err)
		}
		config.DB = db
		logger.Info("database connection initialized", "provider", config.Database.Provider)
	}

	// Apply database connection pool settings to the GORM DB if it's set.
	sqlDB, err := config.DB.DB()
	if err != nil {
		logger.Error("failed to get underlying sql.DB", "error", err)
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(config.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.Database.ConnMaxLifetime)

	return config.DB, nil
}

// -------------------------------
// CONFIG MANAGER
// -------------------------------

// InitConfigManager initializes the appropriate config manager based on mode
func InitConfigManager(config *models.Config) (models.ConfigManager, error) {
	var manager models.ConfigManager
	var err error

	switch config.Mode {
	case models.ModeLibrary:
		{
			slog.Debug("Running in library mode - no config manager needed")
			return nil, nil
		}
	case models.ModeDatabase:
		{
			slog.Info("Initializing database-backed config manager")
			// Init database first
			_, err = InitDatabase(config)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize database for config manager: %w", err)
			}
			manager = internalconfig.NewConfigManager(config)
			// Get config and override
			loadedConfig := manager.GetConfig()
			*config = *loadedConfig
		}
	case models.ModeFile:
		{
			slog.Debug("Initializing file-based config manager")
			manager = internalconfig.NewConfigManager(config)
			// Get config and override
			loadedConfig := manager.GetConfig()
			*config = *loadedConfig
			// Then init database
			_, err = InitDatabase(config)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize database after config load: %w", err)
			}
		}
	default:
		{
			return nil, fmt.Errorf("unsupported mode: %s", config.Mode)
		}
	}

	return manager, nil
}

// -------------------------------
// SECONDARY STORAGE
// -------------------------------

// InitSecondaryStorage wires up the secondary storage implementation based on type
func InitSecondaryStorage(config *models.Config) error {
	storageType := config.SecondaryStorage.Type

	// At the moment currently only supports library mode.
	if storageType == models.SecondaryStorageTypeCustom {
		if config.SecondaryStorage.Storage == nil {
			return fmt.Errorf("custom secondary storage type specified but no storage implementation provided")
		}
		return nil
	}

	switch storageType {
	case models.SecondaryStorageTypeMemory:
		{
			config.SecondaryStorage.Storage = storage.NewMemorySecondaryStorage(config.SecondaryStorage.MemoryOptions)
		}
	case models.SecondaryStorageTypeDatabase:
		{
			if config.DB == nil {
				return fmt.Errorf("database secondary storage type specified but database not initialized")
			}
			config.SecondaryStorage.Storage = storage.NewDatabaseSecondaryStorage(config.DB, config.SecondaryStorage.DatabaseOptions)
		}
	default:
		{
			if config.SecondaryStorage.Storage == nil {
				config.SecondaryStorage.Type = models.SecondaryStorageTypeMemory
				config.SecondaryStorage.Storage = storage.NewMemorySecondaryStorage(config.SecondaryStorage.MemoryOptions)
			}
		}
	}

	return nil
}

// InitEventBus initializes the event bus based on configuration
func InitEventBus(config *models.Config) (models.EventBus, error) {
	var pubsub models.PubSub

	if config.EventBus.PubSub != nil {
		pubsub = config.EventBus.PubSub
		return events.NewEventBus(config, pubsub), nil
	}

	pubsubType := config.EventBus.PubSubType
	if config.EventBus.PubSubType != "" {
		pubsubType = config.EventBus.PubSubType
	}

	switch pubsubType {
	case "memory", "":
		pubsub = events.NewInMemoryPubSub()
	default:
		return nil, fmt.Errorf("unsupported pubsub type: %s (supported: memory)", pubsubType)
	}

	return events.NewEventBus(config, pubsub), nil
}

func InitServices(config *models.Config, eventBus models.EventBus, pluginRateLimits []models.PluginRateLimit) *internalauth.Service {
	userService := services.NewUserServiceImpl(config, config.DB)
	accountService := services.NewAccountServiceImpl(config, config.DB)
	sessionService := services.NewSessionServiceImpl(config, config.DB)
	verificationService := services.NewVerificationServiceImpl(config, config.DB)
	passwordService := services.NewArgon2PasswordService()
	tokenService := services.NewTokenServiceImpl(config)
	rateLimitService := services.NewRateLimitServiceImpl(config, config.Logger.Logger, pluginRateLimits)
	mailerService := services.NewMailerServiceImpl(config)
	webhookExecutor := internalevents.NewWebhookExecutor(config.Logger.Logger)
	eventEmitter := internalevents.NewEventEmitter(config, config.Logger.Logger, eventBus, webhookExecutor)

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

	authService := internalauth.NewService(
		config,
		eventBus,
		webhookExecutor,
		eventEmitter,
		userService,
		accountService,
		sessionService,
		verificationService,
		passwordService,
		tokenService,
		rateLimitService,
		mailerService,
		oauth2ProviderRegistry,
	)

	return authService
}

func InitApi(config *models.Config, authService *internalauth.Service) models.AuthApi {
	useCases := internalauth.NewUseCases(config, authService)

	return internalauth.NewApi(
		*useCases,
		authService,
	)
}

func InitPluginRegistry(config *models.Config, api models.AuthApi, eventBus models.EventBus, apiMiddleware *models.ApiMiddleware) models.PluginRegistry {
	pluginRegistry := plugins.NewPluginRegistry(config, api, eventBus, apiMiddleware)
	for _, p := range config.Plugins.Plugins {
		pluginRegistry.Register(p)
	}
	_ = pluginRegistry.InitAll()

	return pluginRegistry
}
