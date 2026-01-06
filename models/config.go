package models

import (
	"time"
)

// Config holds the core configuration for GoBetterAuth.
type Config struct {
	// Core identity
	AppName  string `json:"app_name" toml:"app_name"`
	BaseURL  string `json:"base_url" toml:"base_url"`
	BasePath string `json:"base_path" toml:"base_path"`
	Secret   string `json:"secret" toml:"secret"`

	// Core infrastructure configuration
	Database DatabaseConfig `json:"database" toml:"database"`
	Logger   LoggerConfig   `json:"logger" toml:"logger"`
	EventBus EventBusConfig `json:"event_bus" toml:"event_bus"`

	// Global trusted origins for CORS, CSRF protection and more
	Security SecurityConfig `json:"security" toml:"security"`

	// Plugin configurations
	Plugins PluginsConfig `json:"plugins" toml:"plugins"`

	// RouteMappings defines plugin-to-route mappings.
	// Each route specifies which plugins should execute hooks for that endpoint.
	// This enables fully declarative plugin routing in both standalone and library modes.
	RouteMappings []RouteMapping `json:"route_mappings" toml:"route_mappings"`

	// PreParsedConfigs stores the original typed plugin config objects.
	// This allows skipping mapstructure unmarshalling and preserving type safety.
	// Key: plugin ID, Value: typed config struct passed to Auth.New()
	PreParsedConfigs map[string]any `json:"-" toml:"-"`
}

type LoggerConfig struct {
	Level string `json:"level" toml:"level"`
}

type DatabaseConfig struct {
	Provider        string        `json:"provider" toml:"provider"`
	URL             string        `json:"url" toml:"url"`
	MaxOpenConns    int           `json:"max_open_conns" toml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" toml:"conn_max_lifetime"`
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

type SecurityConfig struct {
	TrustedOrigins []string `json:"trusted_origins" toml:"trusted_origins"`
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
