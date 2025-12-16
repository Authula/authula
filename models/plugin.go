package models

import (
	"net/http"
)

type PluginMetadata struct {
	Name        string
	Version     string
	Description string
}

// PluginOptions represents dynamic configuration options for a plugin.
type PluginOptions = map[string]any

// PluginConfig holds per-plugin configuration.
type PluginConfig struct {
	Enabled bool
	Options PluginOptions
}

type PluginMiddleware struct {
	Auth          func() func(http.Handler) http.Handler
	OptionalAuth  func() func(http.Handler) http.Handler
	CorsAuth      func() func(http.Handler) http.Handler
	CSRF          func() func(http.Handler) http.Handler
	RateLimit     func() func(http.Handler) http.Handler
	EndpointHooks func() func(http.Handler) http.Handler
}

type PluginContext struct {
	Config     *Config
	Api        *Api
	EventBus   EventBus
	Middleware *PluginMiddleware
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

type Plugin interface {
	Metadata() PluginMetadata
	Config() PluginConfig
	Ctx() *PluginContext
	Init(ctx *PluginContext) error
	Migrations() []any
	Routes() []PluginRoute
	RateLimit() *PluginRateLimit
	DatabaseHooks() *PluginDatabaseHooks
	EventHooks() *PluginEventHooks
	Close() error
}
