package gobetterauth

import (
	"log/slog"
	"os"

	"fmt"

	"gorm.io/gorm"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"

	"github.com/GoBetterAuth/go-better-auth/env"
	"github.com/GoBetterAuth/go-better-auth/events"
	internalauth "github.com/GoBetterAuth/go-better-auth/internal/auth"
	internalconfig "github.com/GoBetterAuth/go-better-auth/internal/config"
	internalevents "github.com/GoBetterAuth/go-better-auth/internal/events"
	"github.com/GoBetterAuth/go-better-auth/internal/plugins"
	"github.com/GoBetterAuth/go-better-auth/internal/services"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	oauth2providers "github.com/GoBetterAuth/go-better-auth/oauth2-providers"
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

	// If DB is already initialized, apply pool settings and return
	if config.DB != nil {
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

	// Check if database config is valid
	if config.Database.Provider == "" {
		return nil, fmt.Errorf("database provider must be specified")
	}

	databaseUrl := os.Getenv(env.DatabaseURL)
	if databaseUrl == "" {
		if config.Database.URL == "" {
			return nil, fmt.Errorf("database connection string must be specified via %s environment variable or config", env.DatabaseURL)
		} else {
			databaseUrl = config.Database.URL
		}
	} else {
		config.Database.URL = databaseUrl
	}

	// Initialize database connection
	var dialector gorm.Dialector
	switch config.Database.Provider {
	case "sqlite":
		dialector = sqlite.Open(databaseUrl)
	case "postgres":
		dialector = postgres.Open(databaseUrl)
	case "mysql":
		dialector = mysql.Open(databaseUrl)
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", config.Database.Provider)
	}

	gormConfig := &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 nil,
	}
	if config.Logger.Level == "debug" {
		gormConfig.Logger = nil // Use GORM's default debug logger
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	config.DB = db
	logger.Info("database connection initialized", "provider", config.Database.Provider)

	// Apply connection pool settings
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

func RunCoreMigrations(db *gorm.DB) {
	dbModels := []any{
		// Admin
		&models.AuthSettings{},
		// Auth
		&models.User{},
		&models.Account{},
		&models.Session{},
		&models.Verification{},
		&models.KeyValueStore{},
	}

	// Auto-migrate core models
	if err := db.AutoMigrate(dbModels...); err != nil {
		slog.Error("failed to auto migrate database", slog.Any("error", err))
		panic(err)
	}
}

func RunPluginMigrations(pluginRegistry models.PluginRegistry) {
	// Auto-migrate plugin models
	if err := pluginRegistry.RunMigrations(); err != nil {
		slog.Error("failed to run plugin migrations", slog.Any("error", err))
		panic(err)
	}
}

// -------------------------------
// CONFIG MANAGER
// -------------------------------

// InitConfigManager initializes the appropriate config manager based on mode
func InitConfigManager(config *models.Config) (models.ConfigManager, error) {
	var manager models.ConfigManager

	switch config.Mode {
	case models.ModeLibrary:
		{
			return nil, nil
		}
	case models.ModeStandalone:
		{
			manager = internalconfig.NewConfigManager(config)
			if err := manager.Init(); err != nil {
				return nil, fmt.Errorf("failed to initialize config manager: %w", err)
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

func InitServices(config *models.Config, configManager models.ConfigManager, eventBus models.EventBus, pluginRateLimits []models.PluginRateLimit) *internalauth.Service {
	userService := services.NewUserServiceImpl(config, config.DB)
	accountService := services.NewAccountServiceImpl(config, config.DB)
	sessionService := services.NewSessionServiceImpl(config, config.DB)
	verificationService := services.NewVerificationServiceImpl(config, config.DB)
	passwordService := services.NewArgon2PasswordService()
	tokenService := services.NewTokenServiceImpl(config)
	rateLimitService := services.NewRateLimitServiceImpl(config, config.Logger.Logger, pluginRateLimits)
	mailerService := services.NewSMTPMailerService(config)
	webhookExecutor := internalevents.NewWebhookExecutor(config.Logger.Logger)
	eventEmitter := internalevents.NewEventEmitter(config, config.Logger.Logger, eventBus, webhookExecutor)
	oauth2ProviderRegistry := oauth2providers.NewOAuth2ProviderRegistry(config)

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

	oauth2ProviderRegistry.RefreshOAuth2Providers()

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
