package models

import (
	"context"
	"embed"
	"net/http"

	"github.com/uptrace/bun"
)

type PluginID string

const (
	PluginConfigManager    PluginID = "config_manager"
	PluginSecondaryStorage PluginID = "secondary_storage"
	PluginEmail            PluginID = "email"
	PluginCSRF             PluginID = "csrf"
	PluginEmailPassword    PluginID = "email_password"
	PluginOAuth2           PluginID = "oauth2"
	PluginSession          PluginID = "session"
	PluginJWT              PluginID = "jwt"
	PluginBearer           PluginID = "bearer"
	PluginRateLimit        PluginID = "ratelimit"
)

func (id PluginID) String() string {
	return string(id)
}

// PluginMetadata contains metadata about a plugin
type PluginMetadata struct {
	ID          string
	Version     string
	Description string
}

// PluginContext is the context passed to plugins during initialization.
type PluginContext struct {
	DB              bun.IDB
	Logger          Logger
	EventBus        EventBus
	ServiceRegistry ServiceRegistry
	GetConfig       func() *Config
}

// Plugin is the base interface all plugins must implement
type Plugin interface {
	Metadata() PluginMetadata
	Config() any
	Init(ctx *PluginContext) error
	Close() error
}

// PluginWithMigrations is an optional interface for plugins that have database migrations
type PluginWithMigrations interface {
	// Migrations returns the embedded SQL migrations filesystem for the given database provider.
	// The dbProvider parameter should be one of: "postgres", "mysql", "sqlite".
	// The returned embed.FS should contain migration files organized as:
	//   - 20250115000001_migration_name.up.sql
	//   - 20250115000001_migration_name.down.sql
	Migrations(ctx context.Context, dbProvider string) (*embed.FS, error)
}

// PluginWithRoutes is an optional interface for plugins that provide HTTP routes
type PluginWithRoutes interface {
	Routes() []Route
}

// PluginWithMiddleware is an optional interface for plugins that provide global middleware
type PluginWithMiddleware interface {
	Middleware() []func(http.Handler) http.Handler
}

// AuthMethodProvider is an interface for plugins that provide authentication mechanisms
type AuthMethodProvider interface {
	AuthMiddleware() func(http.Handler) http.Handler
	OptionalAuthMiddleware() func(http.Handler) http.Handler
}

// MiddlewareProvider is an interface for plugins that provide global middleware
type MiddlewareProvider interface {
	Middleware() func(http.Handler) http.Handler
}

// PluginWithConfigWatcher is an optional interface that plugins can implement
// to receive real-time config updates. When the config is updated in the database,
// the ConfigManager will call OnConfigUpdate with the new config. Plugins should
// use this callback to update their own config structs using ParsePluginConfig,
// which ensures their internal config stays synchronized without changing pointer references.
type PluginWithConfigWatcher interface {
	OnConfigUpdate(config *Config) error
}

// HookStage defines when a hook should be executed in the request lifecycle
type HookStage int

const (
	// HookOnRequest is executed for every request at the very start
	HookOnRequest HookStage = iota
	// HookBefore is executed before route matching and handling
	HookBefore
	// HookAfter is executed after route handling but before response is sent
	HookAfter
	// HookOnResponse is executed after the response has been written
	HookOnResponse
)

// HookMatcher is a function that determines whether a hook should execute
// for a given request context. It allows hooks to be conditionally applied
// based on path, method, headers, or other request properties.
type HookMatcher func(reqCtx *RequestContext) bool

// HookHandler is the function that executes a hook. It receives the request
// context and can modify request state, set UserID, populate Values, or set
// the Handled flag to short-circuit further processing.
type HookHandler func(reqCtx *RequestContext) error

// Hook defines a request lifecycle hook that can be registered by plugins.
// Hooks provide a clean mechanism for plugins to intercept and modify the
// request lifecycle without tight coupling to the router.
//
// Execution Semantics:
//   - Hooks execute in three phases: HookOnRequest → HookBefore (route handling) → HookAfter (before response)
//   - Within each stage, hooks are sorted by PluginID first (grouping), then by Order within each plugin
//   - Order values are plugin-local: comparing order only makes sense between hooks with the same PluginID
//   - If a hook's Handler returns an error, it is logged but does not stop further hook execution
//   - If a hook sets ctx.Handled=true, execution of subsequent hooks at that stage stops
//   - Hooks without a PluginID execute for all routes; hooks with PluginID only execute if listed in route metadata
//   - If a hook's Matcher returns false, the hook is skipped for that request
//   - Errors returned by HookHandler should not panic; they are handled gracefully
//   - Async hooks execute in background goroutines and do not block the response (side-effects only)
type Hook struct {
	// Stage determines when this hook is executed
	Stage HookStage
	// PluginID identifies a plugin capability (e.g., "session.auth", "bearer.auth", "csrf.protect").
	// If set, hook only executes if PluginID is in ctx.Route.Metadata["plugins"].
	// If empty, hook executes based on Matcher logic.
	PluginID string
	// Matcher optionally filters when this hook should run (optional).
	// Checked after PluginID filtering, if present.
	Matcher HookMatcher
	// Handler is the function that executes the hook
	Handler HookHandler
	// Order determines execution order when multiple hooks are at the same stage
	// Lower order values execute first (0 is before 1, which is before 2, etc.).
	// Order is local to the plugin (only compared against other hooks with same PluginID).
	Order int
	// Async determines if this hook runs in a background goroutine without blocking the response.
	// Async hooks are for side-effects only (logging, analytics, events, webhooks, secondary storage).
	// Must be false for all auth validation, CSRF, rate-limiting, and critical security hooks.
	// Async hooks execute with a timeout to prevent leaks and have no access to response writer.
	Async bool
}

// PluginWithHooks is an optional interface that plugins can implement
// to provide request lifecycle hooks.
type PluginWithHooks interface {
	Hooks() []Hook
}

type PluginOption func(p Plugin)

// PluginRegistry manages plugin registration and lifecycle
type PluginRegistry interface {
	Register(p Plugin) error
	InitAll() error
	RunMigrations(ctx context.Context) error
	DropMigrations(ctx context.Context) error
	Plugins() []Plugin
	GetConfig() *Config
	CloseAll()
	GetPlugin(pluginID string) Plugin
}
