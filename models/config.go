package models

import (
	"time"

	"github.com/Authula/authula/events"
)

// Config holds the core configuration for Authula.
type Config struct {
	// Core identity
	AppName       string             `json:"app_name" toml:"app_name"`
	BaseURL       string             `json:"base_url" toml:"base_url"`
	BasePath      string             `json:"base_path" toml:"base_path"`
	Secret        string             `json:"secret" toml:"secret"`
	DisabledPaths []string           `json:"disabled_paths" toml:"disabled_paths"`
	Database      DatabaseConfig     `json:"database" toml:"database"`
	Logger        LoggerConfig       `json:"logger" toml:"logger"`
	Session       SessionConfig      `json:"session" toml:"session"`
	Verification  VerificationConfig `json:"verification" toml:"verification"`
	Security      SecurityConfig     `json:"security" toml:"security"`
	EventBus      EventBusConfig     `json:"event_bus" toml:"event_bus"`
	Plugins       PluginsConfig      `json:"plugins" toml:"plugins"`
	// RouteMappings defines plugin-to-route mappings.
	// Each entry can declare multiple routes in Paths using METHOD:/path strings.
	// A path without a method prefix applies to all HTTP methods.
	// This enables fully declarative plugin routing in both standalone and library modes.
	RouteMappings []RouteMapping `json:"route_mappings" toml:"route_mappings"`
	// PreParsedConfigs stores the original typed plugin config objects.
	// This allows skipping mapstructure unmarshalling and preserving type safety.
	// Key: plugin ID, Value: typed config struct passed to Auth.New()
	PreParsedConfigs map[string]any `json:"-" toml:"-"`
	// CoreServiceHooks allows you to hook into the service layer to carry out custom logic for users, accounts, sessions, and verifications.
	CoreServiceHooks *CoreServiceHooksConfig `json:"-" toml:"-"`
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
	CookieName         string        `json:"cookie_name" toml:"cookie_name"`
	ExpiresIn          time.Duration `json:"expires_in" toml:"expires_in"`         // Sliding window per activity
	UpdateAge          time.Duration `json:"update_age" toml:"update_age"`         // How often to check/update
	CookieMaxAge       time.Duration `json:"cookie_max_age" toml:"cookie_max_age"` // Absolute max age of the cookie
	Secure             bool          `json:"secure" toml:"secure"`
	HttpOnly           bool          `json:"http_only" toml:"http_only"`
	SameSite           string        `json:"same_site" toml:"same_site"`
	AutoCleanup        bool          `json:"auto_cleanup" toml:"auto_cleanup"`
	CleanupInterval    time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
	MaxSessionsPerUser int           `json:"max_sessions_per_user" toml:"max_sessions_per_user"`
}

type VerificationConfig struct {
	AutoCleanup     bool          `json:"auto_cleanup" toml:"auto_cleanup"`
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
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
	Prefix                string                  `json:"prefix" toml:"prefix"`
	MaxConcurrentHandlers int                     `json:"max_concurrent_handlers" toml:"max_concurrent_handlers"`
	ContextTimeout        time.Duration           `json:"context_timeout" toml:"context_timeout"`
	Provider              events.EventBusProvider `json:"provider" toml:"provider"`
	GoChannel             *GoChannelConfig        `json:"go_channel" toml:"go_channel"`
	SQLite                *SQLiteConfig           `json:"sqlite" toml:"sqlite"`
	PostgreSQL            *PostgreSQLConfig       `json:"postgres" toml:"postgres"`
	Redis                 *RedisConfig            `json:"redis" toml:"redis"`
	Kafka                 *KafkaConfig            `json:"kafka" toml:"kafka"`
	NATS                  *NatsConfig             `json:"nats" toml:"nats"`
	RabbitMQ              *RabbitMQConfig         `json:"rabbitmq" toml:"rabbitmq"`
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

// RouteMapping defines which plugins should execute for one or more routes.
// Used in both standalone and library modes to declaratively map route patterns to plugins.
// Standalone: via config.toml [[route_mappings]] table
// Library: via config.RouteMappings or WithRouteMappings option
//
// Example:
//
//	[[route_mappings]]
//	paths = ["GET:/auth/me", "/admin/*"]
//	plugins = ["session.auth", "bearer.auth"]
//	permissions = ["users.read"]
type RouteMapping struct {
	Paths       []string `json:"paths" toml:"paths"`
	Plugins     []string `json:"plugins" toml:"plugins"`
	Permissions []string `json:"permissions" toml:"permissions"`
}

type CoreServiceHooksConfig struct {
	Users         *UserServiceHooks
	Accounts      *AccountServiceHooks
	Sessions      *SessionServiceHooks
	Verifications *VerificationServiceHooks
}

type UserHook func(user *User) error

type AccountHook func(account *Account) error

type SessionHook func(session *Session) error

type VerificationHook func(verification *Verification) error

type UserServiceHooks struct {
	beforeCreate []UserHook
	afterCreate  []UserHook
	beforeUpdate []UserHook
	afterUpdate  []UserHook
}

func (h *UserServiceHooks) RegisterBeforeCreate(fn UserHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *UserServiceHooks) RegisterAfterCreate(fn UserHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *UserServiceHooks) RegisterBeforeUpdate(fn UserHook) {
	h.beforeUpdate = append(h.beforeUpdate, fn)
}

func (h *UserServiceHooks) RegisterAfterUpdate(fn UserHook) {
	h.afterUpdate = append(h.afterUpdate, fn)
}

func (h *UserServiceHooks) BeforeCreateHooks() []UserHook {
	return h.beforeCreate
}

func (h *UserServiceHooks) AfterCreateHooks() []UserHook {
	return h.afterCreate
}

func (h *UserServiceHooks) BeforeUpdateHooks() []UserHook {
	return h.beforeUpdate
}

func (h *UserServiceHooks) AfterUpdateHooks() []UserHook {
	return h.afterUpdate
}

type AccountServiceHooks struct {
	beforeCreate []AccountHook
	afterCreate  []AccountHook
	beforeUpdate []AccountHook
	afterUpdate  []AccountHook
}

func (h *AccountServiceHooks) RegisterBeforeCreate(fn AccountHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *AccountServiceHooks) RegisterAfterCreate(fn AccountHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *AccountServiceHooks) RegisterBeforeUpdate(fn AccountHook) {
	h.beforeUpdate = append(h.beforeUpdate, fn)
}

func (h *AccountServiceHooks) RegisterAfterUpdate(fn AccountHook) {
	h.afterUpdate = append(h.afterUpdate, fn)
}

func (h *AccountServiceHooks) BeforeCreateHooks() []AccountHook {
	return h.beforeCreate
}

func (h *AccountServiceHooks) AfterCreateHooks() []AccountHook {
	return h.afterCreate
}

func (h *AccountServiceHooks) BeforeUpdateHooks() []AccountHook {
	return h.beforeUpdate
}

func (h *AccountServiceHooks) AfterUpdateHooks() []AccountHook {
	return h.afterUpdate
}

type SessionServiceHooks struct {
	beforeCreate []SessionHook
	afterCreate  []SessionHook
	beforeUpdate []SessionHook
	afterUpdate  []SessionHook
}

func (h *SessionServiceHooks) RegisterBeforeCreate(fn SessionHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *SessionServiceHooks) RegisterAfterCreate(fn SessionHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *SessionServiceHooks) RegisterBeforeUpdate(fn SessionHook) {
	h.beforeUpdate = append(h.beforeUpdate, fn)
}

func (h *SessionServiceHooks) RegisterAfterUpdate(fn SessionHook) {
	h.afterUpdate = append(h.afterUpdate, fn)
}

func (h *SessionServiceHooks) BeforeCreateHooks() []SessionHook {
	return h.beforeCreate
}

func (h *SessionServiceHooks) AfterCreateHooks() []SessionHook {
	return h.afterCreate
}

func (h *SessionServiceHooks) BeforeUpdateHooks() []SessionHook {
	return h.beforeUpdate
}

func (h *SessionServiceHooks) AfterUpdateHooks() []SessionHook {
	return h.afterUpdate
}

type VerificationServiceHooks struct {
	beforeCreate []VerificationHook
	afterCreate  []VerificationHook
}

func (h *VerificationServiceHooks) RegisterBeforeCreate(fn VerificationHook) {
	h.beforeCreate = append(h.beforeCreate, fn)
}

func (h *VerificationServiceHooks) RegisterAfterCreate(fn VerificationHook) {
	h.afterCreate = append(h.afterCreate, fn)
}

func (h *VerificationServiceHooks) BeforeCreateHooks() []VerificationHook {
	return h.beforeCreate
}

func (h *VerificationServiceHooks) AfterCreateHooks() []VerificationHook {
	return h.afterCreate
}
