package domain

import (
	"fmt"
	"net/http"
)

type PluginMetadata struct {
	Name        string
	Version     string
	Description string
}

type PluginOptions = map[string]any

// PluginConfig holds per-plugin configuration.
type PluginConfig struct {
	Enabled bool
	Options PluginOptions
}

type PluginContext struct {
	Config   *Config
	EventBus EventBus
}

type PluginRouteMiddleware func(http.Handler) http.Handler
type PluginRouteHandler func() http.Handler

type PluginRoute struct {
	Method     string
	Path       string // Relative path, /auth is auto-prefixed
	Middleware []PluginRouteMiddleware
	Handler    PluginRouteHandler
}

type PluginRateLimit = RateLimitConfig

type BeforeCreateHook[T any] func(ctx *PluginContext, entity *T) error
type AfterCreateHook[T any] func(ctx *PluginContext, entity *T) error

type BeforeReadHook[T any] func(ctx *PluginContext) error
type AfterReadHook[T any] func(ctx *PluginContext, results *[]T) error

type BeforeUpdateHook[T any] func(ctx *PluginContext, existing *T, updatedData map[string]any) error
type AfterUpdateHook[T any] func(ctx *PluginContext, updated *T) error

type BeforeDeleteHook[T any] func(ctx *PluginContext, entity *T) error
type AfterDeleteHook[T any] func(ctx *PluginContext, entity *T) error

type PluginDatabaseHookOperations[T any] struct {
	BeforeCreate *BeforeCreateHook[T]
	AfterCreate  *AfterCreateHook[T]

	BeforeRead *BeforeReadHook[T]
	AfterRead  *AfterReadHook[T]

	BeforeUpdate *BeforeUpdateHook[T]
	AfterUpdate  *AfterUpdateHook[T]

	BeforeDelete *BeforeDeleteHook[T]
	AfterDelete  *AfterDeleteHook[T]
}

type PluginDatabaseHooks map[string]PluginDatabaseHookOperations[any]

type PluginEventHookPayload any

type PluginEventHookFunc func(ctx *PluginContext, payload PluginEventHookPayload) error

type PluginEventHooks map[string]PluginEventHookFunc

type TypedPluginEventHook[T any] func(ctx *PluginContext, payload T) error

// WrapEventHook wraps a typed event hook into a generic PluginEventHookFunc.
func WrapEventHook[T any](hook TypedPluginEventHook[T]) PluginEventHookFunc {
	return func(ctx *PluginContext, payload PluginEventHookPayload) error {
		typedPayload, ok := payload.(T)
		if !ok {
			return fmt.Errorf(
				"invalid event payload type: expected %T, got %T",
				*new(T),
				payload,
			)
		}
		return hook(ctx, typedPayload)
	}
}

type Plugin interface {
	Metadata() PluginMetadata
	Config() PluginConfig
	Init(ctx *PluginContext) error
	Migrations() []any
	Routes() []PluginRoute
	RateLimit() *PluginRateLimit
	DatabaseHooks() *PluginDatabaseHooks
	EventHooks() *PluginEventHooks
	Close() error
}

// BasePlugin provides default implementations for the Plugin interface.
// Plugins can embed this struct and override only what they need.
type BasePlugin struct {
	metadata      PluginMetadata
	config        PluginConfig
	ctx           *PluginContext
	migrations    []any
	routes        []PluginRoute
	rateLimit     *PluginRateLimit
	databaseHooks *PluginDatabaseHooks
	eventHooks    *PluginEventHooks
}

// NewPlugin creates a new BasePlugin with the provided metadata and config.
func NewPlugin(metadata PluginMetadata, config PluginConfig) *BasePlugin {
	return &BasePlugin{
		metadata:      metadata,
		config:        config,
		ctx:           nil,
		migrations:    []any{},
		routes:        []PluginRoute{},
		rateLimit:     &PluginRateLimit{},
		databaseHooks: &PluginDatabaseHooks{},
		eventHooks:    &PluginEventHooks{},
	}
}

func (plugin *BasePlugin) Metadata() PluginMetadata {
	return plugin.metadata
}

func (plugin *BasePlugin) Config() PluginConfig {
	return plugin.config
}

func (plugin *BasePlugin) Init(ctx *PluginContext) error {
	plugin.ctx = ctx
	return nil
}

func (plugin *BasePlugin) Migrations() []any {
	return plugin.migrations
}

func (plugin *BasePlugin) Routes() []PluginRoute {
	return plugin.routes
}

func (plugin *BasePlugin) RateLimit() *PluginRateLimit {
	return plugin.rateLimit
}

func (plugin *BasePlugin) DatabaseHooks() *PluginDatabaseHooks {
	return plugin.databaseHooks
}

func (plugin *BasePlugin) EventHooks() *PluginEventHooks {
	return plugin.eventHooks
}

func (plugin *BasePlugin) Close() error {
	return nil
}
