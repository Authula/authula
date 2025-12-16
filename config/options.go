package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// NewConfig builds a Config using functional options with sensible defaults.
func NewConfig(options ...models.ConfigOption) *models.Config {
	baseURL := os.Getenv("GO_BETTER_AUTH_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	secret := os.Getenv("GO_BETTER_AUTH_SECRET")
	if secret == "" {
		env := os.Getenv("GO_ENV")
		// Must be set in production
		if env == "production" {
			panic("GO_BETTER_AUTH_SECRET environment variable must be set in production")
		}
		// Use default secret for non-production environments
		secret = "go-better-auth-secret-0123456789"
	}

	// Define sensible defaults first
	config := &models.Config{
		AppName:  "GoBetterAuth",
		BaseURL:  baseURL,
		BasePath: "/auth",
		Secret:   secret,
		DB:       nil,
		Database: models.DatabaseConfig{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
		},
		EmailPassword: models.EmailPasswordConfig{
			Enabled:                  false,
			RequireEmailVerification: false,
			MinPasswordLength:        8,
			MaxPasswordLength:        32,
		},
		EmailVerification: models.EmailVerificationConfig{
			AutoSignIn:   false,
			SendOnSignUp: false,
			SendOnSignIn: false,
			ExpiresIn:    1 * time.Hour,
		},
		User: models.UserConfig{
			ChangeEmail: models.ChangeEmailConfig{},
		},
		Session: models.SessionConfig{
			CookieName: "gobetterauth.session_token",
			ExpiresIn:  7 * 24 * time.Hour,
			UpdateAge:  24 * time.Hour,
		},
		CSRF: models.CSRFConfig{
			Enabled:    false,
			CookieName: "gobetterauth_csrf",
			HeaderName: "X-GOBETTERAUTH-CSRF-TOKEN",
			ExpiresIn:  7 * 24 * time.Hour,
		},
		TrustedOrigins: models.TrustedOriginsConfig{},
		SecondaryStorage: models.SecondaryStorageConfig{
			Type: models.SecondaryStorageTypeMemory,
		},
		RateLimit: models.RateLimitConfig{
			Enabled:   false,
			Window:    1 * time.Minute,
			Max:       100,
			Algorithm: models.RateLimitAlgorithmFixedWindow,
			Prefix:    "rate_limit:",
			IP: models.IPConfig{
				Headers: []string{
					"x-forwarded-for",
				},
			},
		},
		EndpointHooks: models.EndpointHooksConfig{},
		DatabaseHooks: models.DatabaseHooksConfig{},
		EventHooks:    models.EventHooksConfig{},
		EventBus:      models.EventBusConfig{},
	}

	// Apply the options
	for _, option := range options {
		option(config)
	}

	// initialize DB using the configured DatabaseConfig if not already set
	if config.DB == nil && config.Database.Provider != "" && config.Database.ConnectionString != "" {
		var err error
		config.DB, err = initDatabase(config.Database)
		if err != nil {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			logger.Error(
				"failed to open database",
				slog.String("provider", config.Database.Provider),
				slog.String("connection_string", config.Database.ConnectionString),
				slog.Any("error", err),
			)
			panic(err)
		}
	}

	// Apply database connection pool settings to the GORM DB if it's set
	if config.DB != nil {
		sqlDB, err := config.DB.DB()
		if err != nil {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			logger.Error(
				"failed to get underlying sql.DB",
				slog.Any("error", err),
			)
			panic(err)
		}
		sqlDB.SetMaxOpenConns(config.Database.MaxOpenConns)
		sqlDB.SetMaxIdleConns(config.Database.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(config.Database.ConnMaxLifetime)
	}

	return config
}

// initDatabase creates a GORM DB connection based on provider.
func initDatabase(dbConfig models.DatabaseConfig) (*gorm.DB, error) {
	switch dbConfig.Provider {
	case "sqlite":
		return gorm.Open(sqlite.Open(dbConfig.ConnectionString), &gorm.Config{})
	case "postgres":
		return gorm.Open(postgres.Open(dbConfig.ConnectionString), &gorm.Config{})
	case "mysql":
		return gorm.Open(mysql.Open(dbConfig.ConnectionString), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", dbConfig.Provider)
	}
}

func WithAppName(name string) models.ConfigOption {
	return func(c *models.Config) {
		c.AppName = name
	}
}

func WithBaseURL(url string) models.ConfigOption {
	return func(c *models.Config) {
		c.BaseURL = url
	}
}

func WithBasePath(path string) models.ConfigOption {
	return func(c *models.Config) {
		c.BasePath = path
	}
}

func WithSecret(secret string) models.ConfigOption {
	return func(c *models.Config) {
		c.Secret = secret
	}
}

func WithDB(db *gorm.DB) models.ConfigOption {
	return func(c *models.Config) {
		c.DB = db
	}
}

func WithDatabase(db models.DatabaseConfig) models.ConfigOption {
	return func(c *models.Config) {
		if db.Provider != "" {
			c.Database.Provider = db.Provider
		}
		if db.ConnectionString != "" {
			c.Database.ConnectionString = db.ConnectionString
		}
		if db.MaxOpenConns != 0 {
			c.Database.MaxOpenConns = db.MaxOpenConns
		}
		if db.MaxIdleConns != 0 {
			c.Database.MaxIdleConns = db.MaxIdleConns
		}
		if db.ConnMaxLifetime != 0 {
			c.Database.ConnMaxLifetime = db.ConnMaxLifetime
		}
	}
}

func WithSecondaryStorage(storage models.SecondaryStorageConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.SecondaryStorage = storage
	}
}

func WithEmailPassword(config models.EmailPasswordConfig) models.ConfigOption {
	return func(c *models.Config) {
		defaults := c.EmailPassword

		if config.Enabled {
			defaults.Enabled = config.Enabled
		}
		if config.MinPasswordLength != 0 {
			defaults.MinPasswordLength = config.MinPasswordLength
		}
		if config.MaxPasswordLength != 0 {
			defaults.MaxPasswordLength = config.MaxPasswordLength
		}
		if config.DisableSignUp {
			defaults.DisableSignUp = config.DisableSignUp
		}
		if config.RequireEmailVerification {
			defaults.RequireEmailVerification = config.RequireEmailVerification
		}
		if config.AutoSignIn {
			defaults.AutoSignIn = config.AutoSignIn
		}
		if config.SendResetPasswordEmail != nil {
			defaults.SendResetPasswordEmail = config.SendResetPasswordEmail
		}
		if config.ResetTokenExpiry != 0 {
			defaults.ResetTokenExpiry = config.ResetTokenExpiry
		}
		if config.Password != nil {
			defaults.Password = config.Password
		}

		c.EmailPassword = defaults
	}
}

func WithEmailVerification(config models.EmailVerificationConfig) models.ConfigOption {
	return func(c *models.Config) {
		defaults := c.EmailVerification

		if config.SendVerificationEmail != nil {
			defaults.SendVerificationEmail = config.SendVerificationEmail
		}
		if config.AutoSignIn {
			defaults.AutoSignIn = config.AutoSignIn
		}
		if config.SendOnSignUp {
			defaults.SendOnSignUp = config.SendOnSignUp
		}
		if config.SendOnSignIn {
			defaults.SendOnSignIn = config.SendOnSignIn
		}
		if config.ExpiresIn != 0 {
			defaults.ExpiresIn = config.ExpiresIn
		}

		c.EmailVerification = defaults
	}
}

func WithUser(userConfig models.UserConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.User = userConfig
	}
}

func WithSession(sessionConfig models.SessionConfig) models.ConfigOption {
	return func(c *models.Config) {
		if sessionConfig.CookieName == "" {
			sessionConfig.CookieName = c.Session.CookieName
		}
		if sessionConfig.ExpiresIn == 0 {
			sessionConfig.ExpiresIn = c.Session.ExpiresIn
		}
		if sessionConfig.UpdateAge == 0 {
			sessionConfig.UpdateAge = c.Session.UpdateAge
		}
		c.Session = sessionConfig
	}
}

func WithCSRF(csrfConfig models.CSRFConfig) models.ConfigOption {
	return func(c *models.Config) {
		if csrfConfig.CookieName == "" {
			csrfConfig.CookieName = c.CSRF.CookieName
		}
		if csrfConfig.HeaderName == "" {
			csrfConfig.HeaderName = c.CSRF.HeaderName
		}
		if csrfConfig.ExpiresIn == 0 {
			csrfConfig.ExpiresIn = c.CSRF.ExpiresIn
		}
		c.CSRF = csrfConfig
	}
}

func WithSocialProviders(socialProvidersConfig models.SocialProvidersConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.SocialProviders = socialProvidersConfig
	}
}

func WithTrustedOrigins(trustedOriginsConfig models.TrustedOriginsConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.TrustedOrigins = trustedOriginsConfig
	}
}

func WithRateLimit(rateLimitConfig models.RateLimitConfig) models.ConfigOption {
	return func(c *models.Config) {
		defaults := c.RateLimit

		if rateLimitConfig.Enabled {
			defaults.Enabled = rateLimitConfig.Enabled
		}
		if rateLimitConfig.Window != 0 {
			defaults.Window = rateLimitConfig.Window
		}
		if rateLimitConfig.Max != 0 {
			defaults.Max = rateLimitConfig.Max
		}
		if rateLimitConfig.Algorithm != "" {
			defaults.Algorithm = rateLimitConfig.Algorithm
		}
		if rateLimitConfig.Prefix != "" {
			defaults.Prefix = rateLimitConfig.Prefix
		}
		if rateLimitConfig.CustomRules != nil {
			defaults.CustomRules = rateLimitConfig.CustomRules
		}
		if len(rateLimitConfig.IP.Headers) != 0 {
			defaults.IP.Headers = rateLimitConfig.IP.Headers
		}

		c.RateLimit = defaults
	}
}

func WithEndpointHooks(endpointHooksConfig models.EndpointHooksConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.EndpointHooks = endpointHooksConfig
	}
}

func WithDatabaseHooks(databaseHooksConfig models.DatabaseHooksConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.DatabaseHooks = databaseHooksConfig
	}
}

func WithEventHooks(eventHooksConfig models.EventHooksConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.EventHooks = eventHooksConfig
	}
}

func WithEventBus(eventBusConfig models.EventBusConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.EventBus = eventBusConfig
	}
}

func WithPlugins(config models.PluginsConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.Plugins = config
	}
}
