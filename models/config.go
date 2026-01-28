package models

import (
	"time"
)

// Config holds the core configuration for GoBetterAuth.
type Config struct {
	// Core identity
	AppName  string         `json:"app_name" toml:"app_name"`
	BaseURL  string         `json:"base_url" toml:"base_url"`
	BasePath string         `json:"base_path" toml:"base_path"`
	Secret   string         `json:"secret" toml:"secret"`
	Database DatabaseConfig `json:"database" toml:"database"`
	Logger   LoggerConfig   `json:"logger" toml:"logger"`
	Session  SessionConfig  `json:"session" toml:"session"`
	Security SecurityConfig `json:"security" toml:"security"`
	EventBus EventBusConfig `json:"event_bus" toml:"event_bus"`
	Plugins  PluginsConfig  `json:"plugins" toml:"plugins"`
	// RouteMappings defines plugin-to-route mappings.
	// Each route specifies which plugins should execute hooks for that endpoint.
	// This enables fully declarative plugin routing in both standalone and library modes.
	RouteMappings []RouteMapping `json:"route_mappings" toml:"route_mappings"`
	// PreParsedConfigs stores the original typed plugin config objects.
	// This allows skipping mapstructure unmarshalling and preserving type safety.
	// Key: plugin ID, Value: typed config struct passed to Auth.New()
	PreParsedConfigs map[string]any `json:"-" toml:"-"`
	// DatabaseHooks allows you to hook into database operations for users, accounts, sessions, and verifications.
	DatabaseHooks *CoreDatabaseHooks `json:"-" toml:"-"`
}

type DatabaseConfig struct {
	Provider        string        `json:"provider" toml:"provider"`
	URL             string        `json:"url" toml:"url"`
	MaxOpenConns    int           `json:"max_open_conns" toml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" toml:"conn_max_lifetime"`
}

type LoggerConfig struct {
	Level string `json:"level" toml:"level"`
}

type SessionConfig struct {
	CookieName   string        `json:"cookie_name" toml:"cookie_name"`
	ExpiresIn    time.Duration `json:"expires_in" toml:"expires_in"`         // Sliding window per activity
	UpdateAge    time.Duration `json:"update_age" toml:"update_age"`         // How often to check/update
	CookieMaxAge time.Duration `json:"cookie_max_age" toml:"cookie_max_age"` // Absolute max age of the cookie
	Secure       bool          `json:"secure" toml:"secure"`
	HttpOnly     bool          `json:"http_only" toml:"http_only"`
	SameSite     string        `json:"same_site" toml:"same_site"`
}

type SecurityConfig struct {
	TrustedOrigins []string   `json:"trusted_origins" toml:"trusted_origins"`
	TrustedHeaders []string   `json:"trusted_headers" toml:"trusted_headers"`
	TrustedProxies []string   `json:"trusted_proxies" toml:"trusted_proxies"`
	CORS           CORSConfig `json:"cors" toml:"cors"`
}

type CORSConfig struct {
	AllowCredentials bool          `json:"allow_credentials" toml:"allow_credentials"`
	AllowedOrigins   []string      `json:"allowed_origins" toml:"allowed_origins"`
	AllowedMethods   []string      `json:"allowed_methods" toml:"allowed_methods"`
	AllowedHeaders   []string      `json:"allowed_headers" toml:"allowed_headers"`
	ExposedHeaders   []string      `json:"exposed_headers" toml:"exposed_headers"`
	MaxAge           time.Duration `json:"max_age" toml:"max_age"`
}

type EventBusConfig struct {
	Prefix                string            `json:"prefix" toml:"prefix"`
	MaxConcurrentHandlers int               `json:"max_concurrent_handlers" toml:"max_concurrent_handlers"`
	Provider              string            `json:"provider" toml:"provider"`
	GoChannel             *GoChannelConfig  `json:"go_channel" toml:"go_channel"`
	SQLite                *SQLiteConfig     `json:"sqlite" toml:"sqlite"`
	PostgreSQL            *PostgreSQLConfig `json:"postgres" toml:"postgres"`
	Redis                 *RedisConfig      `json:"redis" toml:"redis"`
	Kafka                 *KafkaConfig      `json:"kafka" toml:"kafka"`
	NATS                  *NatsConfig       `json:"nats" toml:"nats"`
	RabbitMQ              *RabbitMQConfig   `json:"rabbitmq" toml:"rabbitmq"`
}

type GoChannelConfig struct {
	BufferSize int `json:"buffer_size" toml:"buffer_size"`
}

type SQLiteConfig struct {
	DBPath string `json:"db_path" toml:"db_path"`
}

type PostgreSQLConfig struct {
	URL string `json:"url" toml:"url"`
}

type RedisConfig struct {
	URL           string `json:"url" toml:"url"`
	ConsumerGroup string `json:"consumer_group" toml:"consumer_group"`
}

type KafkaConfig struct {
	Brokers       string `json:"brokers" toml:"brokers"`
	ConsumerGroup string `json:"consumer_group" toml:"consumer_group"`
}

type NatsConfig struct {
	URL string `json:"url" toml:"url"`
}

type RabbitMQConfig struct {
	URL string `json:"url" toml:"url"`
}

type SocialProviderConfig struct {
	Enabled      bool     `json:"enabled" toml:"enabled"`
	ClientID     string   `json:"client_id" toml:"client_id"`
	ClientSecret string   `json:"client_secret" toml:"client_secret"`
	RedirectURL  string   `json:"redirect_url" toml:"redirect_url"`
	Scopes       []string `json:"scopes" toml:"scopes"`
}

// PluginsConfig maps plugin IDs to their configurations
type PluginsConfig map[string]any

// RouteMapping defines which plugins should execute for a specific route.
// Used in both standalone and library modes to declaratively map routes to plugins.
// Standalone: via config.toml [[route_mappings]] table
// Library: via config.RouteMappings or WithRouteMappings option
// Example:
//
//	[[route_mappings]]
//	path = "/auth/me"
//	method = "GET"
//	plugins = ["session.auth", "bearer.auth"]
type RouteMapping struct {
	// Path is the route path (e.g., "/auth/me", "/auth/sign-in")
	Path string `json:"path" toml:"path"`
	// Method is the HTTP method (e.g., "GET", "POST", "PUT", "DELETE")
	Method string `json:"method" toml:"method"`
	// Plugins is the list of plugin IDs that should execute for this route.
	// Plugin IDs follow the format "{plugin_name}.{operation}" (e.g., "session.auth", "csrf.protect")
	Plugins []string `json:"plugins" toml:"plugins"`
}

type CoreDatabaseHooks struct {
	Users         *UserDatabaseHooksConfig
	Accounts      *AccountDatabaseHooksConfig
	Sessions      *SessionDatabaseHooksConfig
	Verifications *VerificationDatabaseHooksConfig
}

type UserDatabaseHooksConfig struct {
	BeforeCreate func(user *User) error
	AfterCreate  func(user User) error
	BeforeUpdate func(user *User) error
	AfterUpdate  func(user User) error
}

type AccountDatabaseHooksConfig struct {
	BeforeCreate func(account *Account) error
	AfterCreate  func(account Account) error
	BeforeUpdate func(account *Account) error
	AfterUpdate  func(account Account) error
}

type SessionDatabaseHooksConfig struct {
	BeforeCreate func(session *Session) error
	AfterCreate  func(session Session) error
	BeforeUpdate func(session *Session) error
	AfterUpdate  func(session Session) error
}

type VerificationDatabaseHooksConfig struct {
	BeforeCreate func(verification *Verification) error
	AfterCreate  func(verification Verification) error
}
