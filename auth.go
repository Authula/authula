package gobetterauth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/GoBetterAuth/go-better-auth/env"
	"github.com/GoBetterAuth/go-better-auth/internal/admin"
	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/handlers"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/storage"
)

type Auth struct {
	Config            *models.Config
	logger            models.Logger
	configManager     models.ConfigManager
	mux               *http.ServeMux
	Service           *auth.Service
	Api               models.AuthApi
	routes            []models.CustomRoute
	middleware        *models.ApiMiddleware
	EventBus          models.EventBus
	pluginRegistry    models.PluginRegistry
	OnRestartRequired func() error
}

// New creates a new Auth instance using the provided config and options.
func New(baseConfig *models.Config) *Auth {
	activeConfig := baseConfig
	InitDefaults(activeConfig)
	logger := activeConfig.Logger.Logger

	if _, err := InitDatabase(activeConfig); err != nil {
		panic(fmt.Sprintf("failed to initialize database: %s", err.Error()))
	}
	// Auto-migrate core models. This is crucial to run here to ensure that the core tables
	// exist before following the next steps as some of these functions rely on core tables to exist.
	RunCoreMigrations(activeConfig.DB)

	configManager, err := InitConfigManager(activeConfig)
	if err != nil {
		logger.Error("Failed to initialize config manager", "error", err)
		panic(err.Error())
	}

	if err := InitSecondaryStorage(activeConfig); err != nil {
		logger.Error("Failed to initialize secondary storage", "error", err)
		panic(err.Error())
	} else {
		if activeConfig.SecondaryStorage.Type == models.SecondaryStorageTypeDatabase {
			if dbStorage, ok := activeConfig.SecondaryStorage.Storage.(*storage.DatabaseSecondaryStorage); ok {
				dbStorage.StartCleanup()
			}
		}
	}

	eventBus, err := InitEventBus(activeConfig)
	if err != nil {
		logger.Error("Failed to initialise event bus", "error", err)
		panic(err.Error())
	}

	mux := http.NewServeMux()

	auth := &Auth{
		Config:        activeConfig,
		logger:        logger,
		configManager: configManager,
		mux:           mux,
		routes:        []models.CustomRoute{},
		EventBus:      eventBus,
	}

	apiKey := os.Getenv(env.AdminApiKey)
	adminAuth := func() func(http.Handler) http.Handler {
		return middleware.AdminAuth(apiKey)
	}
	apiMiddleware := &models.ApiMiddleware{
		// Admin
		AdminAuth: adminAuth,
		// Auth
		Auth:          auth.AuthMiddleware,
		OptionalAuth:  auth.OptionalAuthMiddleware,
		CorsAuth:      auth.CorsAuthMiddleware,
		CSRF:          auth.CSRFMiddleware,
		RateLimit:     auth.RateLimitMiddleware,
		EndpointHooks: auth.EndpointHooksMiddleware,
	}
	auth.middleware = apiMiddleware

	pluginRateLimits := []models.PluginRateLimit{}
	for _, p := range activeConfig.Plugins.Plugins {
		if rateLimit := p.RateLimit(); rateLimit != nil && rateLimit.Enabled {
			pluginRateLimits = append(pluginRateLimits, *rateLimit)
		}
	}

	authService := InitServices(activeConfig, configManager, eventBus, pluginRateLimits)
	auth.Service = authService

	api := InitApi(activeConfig, authService)
	auth.Api = api

	pluginRegistry := InitPluginRegistry(activeConfig, api, eventBus, apiMiddleware)
	auth.pluginRegistry = pluginRegistry

	RunPluginMigrations(pluginRegistry)

	if configManager != nil {
		go auth.watchForConfigChanges()
	}

	return auth
}

// watchForConfigChanges watches for configuration changes and updates the active config.
func (auth *Auth) watchForConfigChanges() {
	ctx := context.Background()
	configChan, err := auth.configManager.Watch(ctx)
	if err != nil {
		auth.logger.Error("Failed to start watching config changes", "error", err)
		return
	}

	for updatedConfig := range configChan {
		if updatedConfig != nil {
			restartRequired := util.RequiresRestart(auth.Config, updatedConfig)

			util.PreserveNonSerializableFieldsOnConfig(auth.Config, updatedConfig)
			*auth.Config = *updatedConfig
			auth.logger.Debug("Configuration updated via watcher")

			if restartRequired {
				auth.logger.Info("Configuration change requires server restart")
				if auth.OnRestartRequired != nil {
					if err := auth.OnRestartRequired(); err != nil {
						auth.logger.Error("Failed to handle restart requirement", "error", err)
					}
				} else {
					auth.logger.Warn("Configuration change requires restart but no restart handler is set")
				}
			}
		}
	}
}

// ---------------------------------
// MIGRATIONS
// ---------------------------------

// RunMigrations is a helper function to run all necessary database migrations for core and plugins manually.
// This is already ran automatically during Auth initialization, so this function is only needed if you want to
// run migrations manually for some reason.
func (auth *Auth) RunMigrations() {
	RunCoreMigrations(auth.Config.DB)
	RunPluginMigrations(auth.pluginRegistry)
}

// DropMigrations is a helper function to drop all database tables related to core and plugins.
// Use with caution as this will delete all data in those tables.
func (auth *Auth) DropMigrations() {
	models := []any{
		// Auth
		&models.KeyValueStore{},
		&models.Verification{},
		&models.Session{},
		&models.Account{},
		&models.User{},
		// Admin
		&models.AuthSettings{},
	}
	for _, model := range models {
		if err := auth.Config.DB.Migrator().DropTable(model); err != nil {
			auth.logger.Error("failed to drop table", slog.Any("model", model), slog.Any("error", err))
			panic(err)
		}
	}

	for _, plugin := range auth.pluginRegistry.Plugins() {
		migrations := plugin.Migrations()
		if len(migrations) == 0 {
			continue
		}

		for _, model := range migrations {
			if err := auth.Config.DB.Migrator().DropTable(model); err != nil {
				auth.logger.Error("failed to drop table", slog.Any("model", model), slog.Any("error", err))
				panic(err)
			}
		}
	}
}

// ---------------------------------
// MIDDLEWARES & HANDLERS
// ---------------------------------

func (auth *Auth) AuthMiddleware() func(http.Handler) http.Handler {
	return middleware.AuthMiddleware(
		auth.Service,
		auth.Config.Session.CookieName,
	)
}

func (auth *Auth) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return middleware.OptionalAuthMiddleware(
		auth.Service,
		auth.Config.Session.CookieName,
	)
}

func (auth *Auth) CorsAuthMiddleware() func(http.Handler) http.Handler {
	return middleware.CorsMiddleware(
		auth.Config.TrustedOrigins.Origins,
	)
}

func (auth *Auth) CSRFMiddleware() func(http.Handler) http.Handler {
	return middleware.CSRFMiddleware(auth.Config.CSRF)
}

func (auth *Auth) RateLimitMiddleware() func(http.Handler) http.Handler {
	return middleware.RateLimitMiddleware(auth.Service.RateLimitService)
}

func (auth *Auth) EndpointHooksMiddleware() func(http.Handler) http.Handler {
	return middleware.EndpointHooksMiddleware(auth.Config, auth.Service)
}

func (auth *Auth) RedirectAuthMiddleware(redirectURL string, status int) func(http.Handler) http.Handler {
	return middleware.RedirectAuthMiddleware(auth.Service, auth.Config.Session.CookieName, redirectURL, status)
}

func (auth *Auth) GetUserIDFromContext(ctx context.Context) (string, bool) {
	value := ctx.Value(middleware.ContextUserID)
	if value == nil {
		return "", false
	}
	id, ok := value.(string)

	return id, ok
}

func (auth *Auth) GetUserIDFromRequest(req *http.Request) (string, bool) {
	return auth.GetUserIDFromContext(req.Context())
}

func (auth *Auth) RegisterRoute(route models.CustomRoute) {
	originalHandler := route.Handler
	route.Handler = func(config *models.Config) http.Handler {
		handler := originalHandler(config)
		finalHandler := handler
		for i := len(route.Middleware) - 1; i >= 0; i-- {
			finalHandler = route.Middleware[i](finalHandler)
		}
		return finalHandler
	}
	auth.routes = append(auth.routes, route)
}

// Handler sets up all routes and returns the final http.Handler
func (auth *Auth) Handler() http.Handler {
	basePath := auth.Config.BasePath

	// Ensure basePath starts with "/" and does not end with "/"
	if basePath[0] != '/' {
		basePath = "/" + basePath
	}
	if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	// Register Admin Routes if configManager is set. Otherwise we're running in Library mode
	if auth.configManager != nil {
		// We purposely set basePath to "" here so that admin routes are always available at /admin/*
		adminRoutes := admin.GetRoutes(auth.Config, auth.configManager, auth.Service, "", auth.middleware)
		auth.registerBaseRoutes(auth.Config, "", adminRoutes)
	}

	// Register Auth Base Routes
	authBaseRoutes := handlers.GetRoutes(auth.Config, auth.Service, basePath, auth.middleware)
	auth.registerBaseRoutes(auth.Config, basePath, authBaseRoutes)

	auth.registerPluginRoutes(basePath)

	// Add catch-all handler for unmatched routes
	auth.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth.logger.Info("Catch-all handler triggered", "method", r.Method, "path", r.URL.Path)
		util.JSONResponse(w, http.StatusNotFound, map[string]any{"message": "Endpoint not found"})
	}))

	var finalHandler http.Handler = auth.mux
	finalHandler = middleware.EndpointHooksMiddleware(auth.Config, auth.Service)(finalHandler)
	if auth.Config.RateLimit.Enabled {
		finalHandler = auth.RateLimitMiddleware()(finalHandler)
	}

	return finalHandler
}

// registerBaseRoutes registers base routes
func (auth *Auth) registerBaseRoutes(config *models.Config, basePath string, routes []models.CustomRoute) {
	for _, route := range routes {
		path := fmt.Sprintf("%s%s", basePath, route.Path)
		handler := route.Handler(config)

		for i := len(route.Middleware) - 1; i >= 0; i-- {
			handler = route.Middleware[i](handler)
		}

		auth.mux.Handle(fmt.Sprintf("%s %s", route.Method, path), handler)
	}
}

// registerPluginRoutes registers routes from plugins
func (auth *Auth) registerPluginRoutes(basePath string) {
	if auth.pluginRegistry == nil {
		return
	}

	plugins := auth.pluginRegistry.Plugins()
	if len(plugins) == 0 {
		return
	}

	for _, plugin := range plugins {
		pluginRoutes := plugin.Routes()
		if len(pluginRoutes) == 0 {
			continue
		}

		for _, route := range pluginRoutes {
			path := fmt.Sprintf("%s%s", basePath, route.Path)
			handler := route.Handler()

			for i := len(route.Middleware) - 1; i >= 0; i-- {
				handler = route.Middleware[i](handler)
			}

			auth.mux.Handle(fmt.Sprintf("%s %s", route.Method, path), handler)
		}
	}
}

// ClosePlugins calls Close for all registered plugins
func (auth *Auth) ClosePlugins() error {
	if auth.pluginRegistry == nil {
		return nil
	}

	auth.pluginRegistry.CloseAll()

	return nil
}
