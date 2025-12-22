package config

import (
	"os"
	"time"

	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// NewConfig builds a Config using functional options with sensible defaults.
func NewConfig(options ...models.ConfigOption) *models.Config {
	// Define sensible defaults first
	config := &models.Config{
		// Default to library mode
		Mode:     models.ModeLibrary,
		AppName:  "GoBetterAuth",
		BasePath: "/auth",
		Secret:   "",
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
		EventBus: models.EventBusConfig{
			Enabled:               false,
			MaxConcurrentHandlers: 10,
		},
	}

	// Apply the options
	for _, option := range options {
		option(config)
	}

	// Environment variable overrides

	// Set BaseURL
	if baseURL := os.Getenv("GO_BETTER_AUTH_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	} else if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8080"
	}

	// Set Secret
	if secret := os.Getenv("GO_BETTER_AUTH_SECRET"); secret != "" {
		config.Secret = secret
	} else if config.Secret == "" {
		if os.Getenv("GO_ENV") == "production" {
			panic("GO_BETTER_AUTH_SECRET environment variable must be set in production")
		}
		config.Secret = "go-better-auth-secret-0123456789"
	}

	return config
}

func WithMode(mode models.Mode) models.ConfigOption {
	return func(c *models.Config) {
		c.Mode = mode
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

func WithLogger(config models.LoggerConfig) models.ConfigOption {
	return func(c *models.Config) {
		defaults := c.Logger

		if config.Level != "" {
			defaults.Level = config.Level
		}
		if config.Logger != nil {
			defaults.Logger = config.Logger
		}

		c.Logger = defaults
	}
}

func WithDB(db *gorm.DB) models.ConfigOption {
	return func(c *models.Config) {
		c.DB = db
	}
}

func WithDatabase(config models.DatabaseConfig) models.ConfigOption {
	return func(c *models.Config) {
		defaults := c.Database

		if config.Provider != "" {
			defaults.Provider = config.Provider
		}
		if config.ConnectionString != "" {
			defaults.ConnectionString = config.ConnectionString
		}
		if config.MaxOpenConns != 0 {
			defaults.MaxOpenConns = config.MaxOpenConns
		}
		if config.MaxIdleConns != 0 {
			defaults.MaxIdleConns = config.MaxIdleConns
		}
		if config.ConnMaxLifetime != 0 {
			defaults.ConnMaxLifetime = config.ConnMaxLifetime
		}

		c.Database = defaults
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
		if config.Password.Hash != nil {
			defaults.Password.Hash = config.Password.Hash
		}
		if config.Password.Verify != nil {
			defaults.Password.Verify = config.Password.Verify
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
		defaults := c.EventBus

		if eventBusConfig.Enabled {
			defaults.Enabled = eventBusConfig.Enabled
		}
		if eventBusConfig.Prefix != "" {
			defaults.Prefix = eventBusConfig.Prefix
		}
		if eventBusConfig.MaxConcurrentHandlers != 0 {
			defaults.MaxConcurrentHandlers = eventBusConfig.MaxConcurrentHandlers
		}
		if eventBusConfig.PubSubType != "" {
			defaults.PubSubType = eventBusConfig.PubSubType
		}
		if eventBusConfig.PubSub != nil {
			defaults.PubSub = eventBusConfig.PubSub
		}

		c.EventBus = defaults
	}
}

func WithPlugins(config models.PluginsConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.Plugins = config
	}
}

func WithWebhooks(config models.WebhooksConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.Webhooks = config
	}
}

// WithEmailConfig sets the email configuration for sending emails in standalone mode
func WithEmailConfig(emailConfig models.EmailConfig) models.ConfigOption {
	return func(c *models.Config) {
		c.Email = emailConfig
	}
}
