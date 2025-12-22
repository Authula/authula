package models

import (
	"net/http"
)

type PluginMetadata struct {
	Name        string
	Version     string
	Description string
}

// PluginConfig holds per-plugin configuration.
type PluginConfig struct {
	Enabled bool
	Options any
}

type PluginContext struct {
	Config          *Config
	Api             AuthApi
	EventBus        EventBus
	Middleware      *ApiMiddleware
	WebhookExecutor WebhookExecutor
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

type Plugin interface {
	Metadata() PluginMetadata
	SetMetadata(meta PluginMetadata)

	Config() PluginConfig
	SetConfig(cfg PluginConfig)

	Ctx() *PluginContext
	SetCtx(ctx *PluginContext)

	Init(ctx *PluginContext) error
	SetInit(fn func(ctx *PluginContext) error)

	Migrations() []any
	SetMigrations(migrations []any)

	Routes() []PluginRoute
	SetRoutes(routes []PluginRoute)

	RateLimit() *PluginRateLimit
	SetRateLimit(rateLimit *PluginRateLimit)

	EndpointHooks() any
	SetEndpointHooks(hooks any)

	DatabaseHooks() any
	SetDatabaseHooks(hooks any)

	EventHooks() any
	SetEventHooks(hooks any)

	Webhooks() any
	SetWebhooks(hooks any)

	Close() error
	SetClose(fn func() error)
}

type PluginOption func(p Plugin)

type BasePlugin struct {
	metadata      PluginMetadata
	config        PluginConfig
	ctx           *PluginContext
	init          func(ctx *PluginContext) error
	migrations    []any // Database migration structs (GORM models)
	routes        []PluginRoute
	rateLimit     *PluginRateLimit
	endpointHooks any
	databaseHooks any
	eventHooks    any
	webhooks      any
	close         func() error
}

func (p *BasePlugin) Metadata() PluginMetadata {
	return p.metadata
}

func (p *BasePlugin) SetMetadata(meta PluginMetadata) {
	p.metadata = meta
}

func (p *BasePlugin) Config() PluginConfig {
	return p.config
}

func (p *BasePlugin) SetConfig(config PluginConfig) {
	p.config = config
}

func (p *BasePlugin) Ctx() *PluginContext {
	return p.ctx
}

func (p *BasePlugin) SetCtx(ctx *PluginContext) {
	p.ctx = ctx
}

func (p *BasePlugin) Init(ctx *PluginContext) error {
	if p.init != nil {
		return p.init(ctx)
	}
	return nil
}

func (p *BasePlugin) SetInit(fn func(ctx *PluginContext) error) {
	p.init = fn
}

func (p *BasePlugin) Migrations() []any {
	return p.migrations
}

func (p *BasePlugin) SetMigrations(migrations []any) {
	p.migrations = migrations
}

func (p *BasePlugin) Routes() []PluginRoute {
	return p.routes
}

func (p *BasePlugin) SetRoutes(routes []PluginRoute) {
	p.routes = routes
}

func (p *BasePlugin) RateLimit() *PluginRateLimit {
	return p.rateLimit
}

func (p *BasePlugin) SetRateLimit(rateLimit *PluginRateLimit) {
	p.rateLimit = rateLimit
}

func (p *BasePlugin) EndpointHooks() any {
	return p.endpointHooks
}

func (p *BasePlugin) SetEndpointHooks(hooks any) {
	p.endpointHooks = hooks
}

func (p *BasePlugin) DatabaseHooks() any {
	return p.databaseHooks
}

func (p *BasePlugin) SetDatabaseHooks(hooks any) {
	p.databaseHooks = hooks
}

func (p *BasePlugin) EventHooks() any {
	return p.eventHooks
}

func (p *BasePlugin) SetEventHooks(hooks any) {
	p.eventHooks = hooks
}

func (p *BasePlugin) Webhooks() any {
	return p.webhooks
}

func (p *BasePlugin) SetWebhooks(hooks any) {
	p.webhooks = hooks
}

func (p *BasePlugin) Close() error {
	if p.close != nil {
		return p.close()
	}
	return nil
}

func (p *BasePlugin) SetClose(fn func() error) {
	p.close = fn
}

type PluginRegistry interface {
	Register(p Plugin)
	InitAll() error
	RunMigrations() error
	Plugins() []Plugin
	CloseAll()
}
