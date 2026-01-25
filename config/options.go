package config

import (
	"fmt"
	"os"
	"time"

	"github.com/GoBetterAuth/go-better-auth/env"
	"github.com/GoBetterAuth/go-better-auth/models"
)

const defaultSecret = "go-better-auth-secret-0123456789"

type ConfigOption func(*models.Config)

// NewConfig builds a Config using functional options with sensible defaults.
// Panics if event bus configuration is invalid or if required secrets are missing in production.
func NewConfig(options ...ConfigOption) *models.Config {
	// Define sensible defaults first
	config := &models.Config{
		AppName:  "GoBetterAuth",
		BaseURL:  "http://localhost:8080",
		BasePath: "/auth",
		Secret:   defaultSecret,
		Session: models.SessionConfig{
			CookieName: "gobetterauth.session_token",
			ExpiresIn:  time.Hour * 24 * 7, // 7 days by default
			UpdateAge:  time.Hour * 24,     // 24 hours update interval
			Secure:     false,
			HttpOnly:   true,
			SameSite:   "lax",
		},
		Security: models.SecurityConfig{
			TrustedOrigins: []string{},
			CORS: models.CORSConfig{
				AllowCredentials: true,
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"OPTIONS", "GET", "POST", "PUT", "PATCH", "DELETE"},
				AllowedHeaders:   []string{"Authorization", "Content-Type", "Cookie", "Set-Cookie"},
				ExposedHeaders:   []string{},
				MaxAge:           24 * time.Hour,
			},
		},
		Database: models.DatabaseConfig{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Minute * 10,
		},
		Logger:           models.LoggerConfig{},
		EventBus:         models.EventBusConfig{},
		Plugins:          models.PluginsConfig{},
		RouteMappings:    []models.RouteMapping{},
		PreParsedConfigs: make(map[string]any),
		DatabaseHooks:    nil,
	}

	// Apply the options - they override defaults only if non-zero/non-empty
	for _, option := range options {
		option(config)
	}

	// Validate BasePath format
	if config.BasePath != "" && config.BasePath[0] != '/' {
		panic(fmt.Errorf("BasePath must start with '/', got: %q", config.BasePath))
	}

	// Validate event bus configuration
	if err := validateEventBusConfig(&config.EventBus); err != nil {
		panic(fmt.Errorf("invalid event bus configuration: %w", err))
	}

	// Validate production configuration
	if os.Getenv(env.EnvGoEnvironment) == "production" && config.Secret == defaultSecret {
		panic(fmt.Errorf("a custom secret must be set in production mode. Please set a custom secret via configuration or the %s environment variable", env.EnvSecret))
	}

	return config
}

func WithAppName(name string) ConfigOption {
	return func(c *models.Config) {
		if name != "" {
			c.AppName = name
		}
	}
}

func WithBaseURL(url string) ConfigOption {
	return func(c *models.Config) {
		if envValue := os.Getenv(env.EnvBaseURL); envValue != "" {
			c.BaseURL = envValue
		} else if url != "" {
			c.BaseURL = url
		}
	}
}

func WithBasePath(path string) ConfigOption {
	return func(c *models.Config) {
		if path != "" {
			c.BasePath = path
		}
	}
}

func WithSecret(secret string) ConfigOption {
	return func(c *models.Config) {
		if envValue := os.Getenv(env.EnvSecret); envValue != "" {
			c.Secret = envValue
		} else if secret != "" {
			c.Secret = secret
		}
	}
}

func WithSession(config models.SessionConfig) ConfigOption {
	return func(c *models.Config) {
		if config.CookieName != "" {
			c.Session.CookieName = config.CookieName
		}
		if config.ExpiresIn != 0 {
			c.Session.ExpiresIn = config.ExpiresIn
		}
		if config.UpdateAge != 0 {
			c.Session.UpdateAge = config.UpdateAge
		}
		if config.CookieMaxAge != 0 {
			c.Session.CookieMaxAge = config.CookieMaxAge
		}
		c.Session.Secure = config.Secure
		c.Session.HttpOnly = config.HttpOnly
		if config.SameSite != "" {
			c.Session.SameSite = config.SameSite
		}
	}
}

func WithSecurity(config models.SecurityConfig) ConfigOption {
	return func(c *models.Config) {
		if len(config.TrustedOrigins) > 0 {
			c.Security.TrustedOrigins = config.TrustedOrigins
		}
		if len(config.CORS.AllowedOrigins) > 0 {
			c.Security.CORS.AllowedOrigins = config.CORS.AllowedOrigins
		}
		if len(config.CORS.AllowedMethods) > 0 {
			c.Security.CORS.AllowedMethods = config.CORS.AllowedMethods
		}
		if len(config.CORS.AllowedHeaders) > 0 {
			c.Security.CORS.AllowedHeaders = config.CORS.AllowedHeaders
		}
		if len(config.CORS.ExposedHeaders) > 0 {
			c.Security.CORS.ExposedHeaders = config.CORS.ExposedHeaders
		}
		c.Security.CORS.AllowCredentials = config.CORS.AllowCredentials
		if config.CORS.MaxAge != 0 {
			c.Security.CORS.MaxAge = config.CORS.MaxAge
		}
	}
}

func WithDatabase(config models.DatabaseConfig) ConfigOption {
	return func(c *models.Config) {
		if config.Provider != "" {
			c.Database.Provider = config.Provider
		}
		if envValue := os.Getenv(env.EnvDatabaseURL); envValue != "" {
			c.Database.URL = envValue
		} else if config.URL != "" {
			c.Database.URL = config.URL
		}
		if config.MaxOpenConns != 0 {
			c.Database.MaxOpenConns = config.MaxOpenConns
		}
		if config.MaxIdleConns != 0 {
			c.Database.MaxIdleConns = config.MaxIdleConns
		}
		if config.ConnMaxLifetime != 0 {
			c.Database.ConnMaxLifetime = config.ConnMaxLifetime
		}
	}
}

func WithLogger(config models.LoggerConfig) ConfigOption {
	return func(c *models.Config) {
		if config.Level != "" {
			c.Logger.Level = config.Level
		}
	}
}

func WithEventBus(config models.EventBusConfig) ConfigOption {
	return func(c *models.Config) {
		if config.MaxConcurrentHandlers > 0 {
			c.EventBus.MaxConcurrentHandlers = config.MaxConcurrentHandlers
		}
		if config.Provider != "" {
			c.EventBus.Provider = config.Provider
		}
		if config.GoChannel != nil {
			c.EventBus.GoChannel = config.GoChannel
		}
		if config.SQLite != nil {
			c.EventBus.SQLite = config.SQLite
		}
		if config.PostgreSQL != nil {
			c.EventBus.PostgreSQL = config.PostgreSQL
		}
		if config.Redis != nil {
			c.EventBus.Redis = config.Redis
		}
		if config.Kafka != nil {
			c.EventBus.Kafka = config.Kafka
		}
		if config.NATS != nil {
			c.EventBus.NATS = config.NATS
		}
		if config.RabbitMQ != nil {
			c.EventBus.RabbitMQ = config.RabbitMQ
		}
	}
}

func WithPlugins(config models.PluginsConfig) ConfigOption {
	return func(c *models.Config) {
		c.Plugins = config
	}
}

func WithRouteMappings(config []models.RouteMapping) ConfigOption {
	return func(c *models.Config) {
		c.RouteMappings = config
	}
}

func WithDatabaseHooks(config *models.CoreDatabaseHooks) ConfigOption {
	return func(c *models.Config) {
		c.DatabaseHooks = config
	}
}

// validateEventBusConfig validates that the event bus provider has the correct configuration
func validateEventBusConfig(config *models.EventBusConfig) error {
	provider := config.Provider
	if provider == "" {
		provider = "gochannel"
	}

	// Validate that the selected provider has the correct config
	switch provider {
	case "gochannel":
		if config.GoChannel == nil {
			return fmt.Errorf("gochannel provider selected but gochannel config is missing")
		}

	case "sqlite":
		if config.SQLite == nil {
			return fmt.Errorf("sqlite provider selected but sqlite config is missing")
		}

	case "postgres":
		if config.PostgreSQL == nil {
			return fmt.Errorf("postgres provider selected but postgres config is missing")
		}
		if os.Getenv(env.EnvPostgresURL) == "" && config.PostgreSQL.URL == "" {
			return fmt.Errorf("postgres provider selected but postgres.url is empty and POSTGRES_URL env var is not set")
		}

	case "redis":
		if config.Redis == nil {
			return fmt.Errorf("redis provider selected but redis config is missing")
		}
		if os.Getenv(env.EnvRedisURL) == "" && config.Redis.URL == "" {
			return fmt.Errorf("redis provider selected but redis.url is empty and REDIS_URL env var is not set")
		}

	case "kafka":
		if config.Kafka == nil {
			return fmt.Errorf("kafka provider selected but kafka config is missing")
		}
		if config.Kafka.Brokers == "" {
			return fmt.Errorf("kafka provider selected but kafka.brokers is empty")
		}

	case "nats":
		if config.NATS == nil {
			return fmt.Errorf("nats provider selected but nats config is missing")
		}
		if os.Getenv(env.EnvNatsURL) == "" && config.NATS.URL == "" {
			return fmt.Errorf("nats provider selected but nats.url is empty and NATS_URL env var is not set")
		}

	case "rabbitmq":
		if config.RabbitMQ == nil {
			return fmt.Errorf("rabbitmq provider selected but rabbitmq config is missing")
		}
		if os.Getenv(env.EnvRabbitMQURL) == "" && config.RabbitMQ.URL == "" {
			return fmt.Errorf("rabbitmq provider selected but rabbitmq.url is empty and RABBITMQ_URL env var is not set")
		}
	}

	return nil
}
