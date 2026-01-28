package gobetterauth

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/internal"
	"github.com/GoBetterAuth/go-better-auth/internal/plugins"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	coreservices "github.com/GoBetterAuth/go-better-auth/services"
)

type AuthConfig struct {
	Config  *models.Config
	Plugins []models.Plugin
}

// Auth is a composition root and entry point for the authentication framework.
type Auth struct {
	config          *models.Config
	logger          models.Logger
	db              bun.IDB
	router          *Router
	ServiceRegistry models.ServiceRegistry
	PluginRegistry  models.PluginRegistry
	handlerOnce     sync.Once
	coreServices    *coreservices.CoreServices
	Api             internal.CoreAPI
}

// New creates a new Auth instance using the provided config and plugins.
// Handles plugin registration, migrations, and initialization in unified way.
// Works identically whether plugins are manually instantiated or built from config.
func New(authConfig *AuthConfig) *Auth {
	util.InitValidator()

	logger := InitLogger(authConfig.Config)

	db, err := InitDatabase(authConfig.Config, logger, authConfig.Config.Logger.Level)
	if err != nil {
		panic(fmt.Errorf("failed to initialize database: %w", err))
	}

	RunCoreMigrations(context.Background(), logger, authConfig.Config.Database.Provider, db)

	router := NewRouter(authConfig.Config, logger, nil)

	eventBus, err := InitEventBus(authConfig.Config)
	if err != nil {
		panic(fmt.Errorf("failed to initialize event bus: %w", err))
	}

	serviceRegistry := plugins.NewServiceRegistry()

	coreServices := InitCoreServices(authConfig.Config, db, serviceRegistry)

	pluginRegistry := plugins.NewPluginRegistry(
		authConfig.Config,
		logger,
		db,
		serviceRegistry,
		eventBus,
	)

	// Initialize PreParsedConfigs if not already done
	if authConfig.Config.PreParsedConfigs == nil {
		authConfig.Config.PreParsedConfigs = make(map[string]any)
	}

	// Initialize Plugins map if not already done
	if authConfig.Config.Plugins == nil {
		authConfig.Config.Plugins = make(models.PluginsConfig)
	}

	// Cache type-safe configs for all plugins and auto-enable those not explicitly disabled
	for _, plugin := range authConfig.Plugins {
		pluginID := plugin.Metadata().ID
		pluginConfig := plugin.Config()

		// Initialize config map if not exists
		if authConfig.Config.Plugins[pluginID] == nil {
			authConfig.Config.Plugins[pluginID] = make(map[string]any)
		}

		// Check if plugin is explicitly disabled in config
		isDisabled := false
		if pluginConfig, ok := authConfig.Config.Plugins[pluginID]; ok {
			if configMap, ok := pluginConfig.(map[string]any); ok {
				if enabled, found := configMap["enabled"]; found {
					if b, ok := enabled.(bool); ok && !b {
						isDisabled = true
					}
				}
			}
		}

		// If not explicitly disabled and no enabled setting exists, set from plugin config
		if !isDisabled {
			if configMap, ok := authConfig.Config.Plugins[pluginID].(map[string]any); ok {
				if _, hasEnabled := configMap["enabled"]; !hasEnabled {
					configMap["enabled"] = getEnabledFromConfig(pluginConfig)
				}
			}
		}

		// Cache the type-safe config
		authConfig.Config.PreParsedConfigs[pluginID] = pluginConfig
	}

	for _, plugin := range authConfig.Plugins {
		if err := pluginRegistry.Register(plugin); err != nil {
			panic(fmt.Errorf("failed to register plugin: %w", err))
		}
	}

	// Run plugin migrations
	if err := pluginRegistry.RunMigrations(context.Background()); err != nil {
		logger.Error("failed to run plugin migrations", "error", err)
		panic(fmt.Errorf("failed to run plugin migrations: %w", err))
	}

	// Initialize all plugins in order
	// Plugin ordering is managed by the plugin registry to ensure dependencies are satisfied
	// (e.g., Core initializes before ConfigManager, ConfigManager before others)
	if err := pluginRegistry.InitAll(); err != nil {
		logger.Error("failed to initialize plugins", "error", err)
		panic(fmt.Errorf("failed to initialize plugins: %w", err))
	}

	api := internal.NewCoreAPI(logger, coreServices.UserService, coreServices.SessionService)

	auth := &Auth{
		config:          authConfig.Config,
		logger:          logger,
		db:              db,
		router:          router,
		ServiceRegistry: serviceRegistry,
		PluginRegistry:  pluginRegistry,
		coreServices:    coreServices,
		Api:             api,
	}

	// Register middleware NOW (before any routes are registered)
	// This ensures Chi's requirement that all middleware is defined before routes
	auth.registerMiddleware()

	return auth
}

// getEnabledFromConfig tries to get Enabled field from plugin config
func getEnabledFromConfig(config any) bool {
	// Default to enabled if we can't determine the value
	if config == nil {
		return true
	}

	// Use existing util.ParsePluginConfig to convert struct to map
	var configMap map[string]any
	if err := util.ParsePluginConfig(config, &configMap); err != nil {
		return true
	}

	if enabled, ok := configMap["enabled"].(bool); ok {
		return enabled
	}

	// Default to true if not found
	return true
}

func (auth *Auth) RunCoreMigrations(ctx context.Context) error {
	return RunCoreMigrations(ctx, auth.logger, auth.config.Database.Provider, auth.db)
}

func (auth *Auth) DropCoreMigrations(ctx context.Context) error {
	return DropCoreMigrations(ctx, auth.logger, auth.config.Database.Provider, auth.db)
}

// registerMiddleware registers all middleware from hooks and plugins
func (auth *Auth) registerMiddleware() {
	currentConfig := auth.PluginRegistry.GetConfig()

	// Register Plugin Global Middleware
	for _, plugin := range auth.PluginRegistry.Plugins() {
		pluginID := plugin.Metadata().ID

		if !util.IsPluginEnabled(currentConfig, pluginID, false) {
			auth.logger.Debug("skipping disabled plugin", "plugin", pluginID)
			continue
		}

		if middlewareProvider, ok := plugin.(models.PluginWithMiddleware); ok {
			middleware := middlewareProvider.Middleware()
			if len(middleware) == 0 {
				auth.logger.Debug("no middleware functions returned", "plugin", pluginID)
				continue
			}
			auth.router.RegisterMiddleware(middleware...)
		}
	}
}

// RegisterMiddleware registers middleware to the chi router.
// Middleware should be registered before calling Handler().
func (auth *Auth) RegisterMiddleware(middleware ...func(http.Handler) http.Handler) {
	auth.router.RegisterMiddleware(middleware...)
}

// RegisterRoute registers a single route with the basePath prefix
func (auth *Auth) RegisterRoute(route models.Route) {
	auth.router.RegisterRoute(route)
}

// RegisterRoutes registers multiple routes with the basePath prefix
func (auth *Auth) RegisterRoutes(routes []models.Route) {
	auth.router.RegisterRoutes(routes)
}

// RegisterCustomRoute registers a single custom route without the basePath prefix
// This is useful for application routes that should not be under the auth basePath
func (auth *Auth) RegisterCustomRoute(route models.Route) {
	auth.router.RegisterCustomRoute(route)
}

// RegisterCustomRoutes registers multiple custom routes without the basePath prefix
// This is useful for application routes that should not be under the auth basePath
func (auth *Auth) RegisterCustomRoutes(routes []models.Route) {
	auth.router.RegisterCustomRoutes(routes)
}

// RegisterHook registers a single hook to the router.
// Hooks allow developers to intercept and modify requests/responses at various stages
// of the request lifecycle (OnRequest, Before, After, OnResponse).
func (auth *Auth) RegisterHook(hook models.Hook) {
	auth.router.RegisterHook(hook)
}

// RegisterHooks registers multiple hooks to the router.
// Hooks allow developers to intercept and modify requests/responses at various stages
// of the request lifecycle (OnRequest, Before, After, OnResponse).
func (auth *Auth) RegisterHooks(hooks []models.Hook) {
	auth.router.RegisterHooks(hooks)
}

// GetUserIDFromContext retrieves the user ID from a context.
// Returns the user ID and a boolean indicating whether it was found.
// This is a convenience wrapper around models.GetUserIDFromContext to avoid
// requiring application code to import the models package.
func (auth *Auth) GetUserIDFromContext(ctx context.Context) (string, bool) {
	return models.GetUserIDFromContext(ctx)
}

// GetUserIDFromRequest retrieves the user ID from an HTTP request's context.
// Returns the user ID and a boolean indicating whether it was found.
// This is a convenience wrapper around models.GetUserIDFromRequest to avoid
// requiring application code to import the models package.
func (auth *Auth) GetUserIDFromRequest(req *http.Request) (string, bool) {
	return models.GetUserIDFromRequest(req)
}

// Handler returns the HTTP handler that serves all authentication routes and hooks.
// It registers routes and hooks from all plugins with the router.
// This is the entry point for HTTP traffic.
func (auth *Auth) Handler() http.Handler {
	auth.handlerOnce.Do(func() {
		auth.router.RegisterRoutes(
			internal.CoreRoutes(
				auth.logger,
				auth.coreServices.UserService,
				auth.coreServices.SessionService,
			),
		)

		currentConfig := auth.config

		// Convert route mappings to metadata format for plugin routing
		// This works identically whether RouteMappings come from config file or library mode
		if len(currentConfig.RouteMappings) > 0 {
			routeMetadata, err := util.ConvertRouteMetadata(currentConfig.RouteMappings)
			if err != nil {
				auth.logger.Error("failed to convert route metadata", "error", err)
			} else {
				adjustedMetadata := make(map[string]map[string]any)
				for key, metadata := range routeMetadata {
					adjustedKey := util.ApplyBasePathToMetadataKey(key, auth.router.basePath)
					adjustedMetadata[adjustedKey] = metadata
				}
				auth.router.SetRouteMetadataFromConfig(adjustedMetadata)
			}
		}

		// Register Plugin Routes
		for _, plugin := range auth.PluginRegistry.Plugins() {
			if !util.IsPluginEnabled(currentConfig, plugin.Metadata().ID, false) {
				continue
			}
			if routeProvider, ok := plugin.(models.PluginWithRoutes); ok {
				pluginRoutes := routeProvider.Routes()
				if len(pluginRoutes) == 0 {
					continue
				}
				auth.router.RegisterRoutes(pluginRoutes)
			}
		}

		// Register Plugin Hooks
		for _, plugin := range auth.PluginRegistry.Plugins() {
			if !util.IsPluginEnabled(currentConfig, plugin.Metadata().ID, false) {
				continue
			}
			if hookProvider, ok := plugin.(models.PluginWithHooks); ok {
				hooks := hookProvider.Hooks()
				if len(hooks) > 0 {
					auth.router.RegisterHooks(hooks)
				}
			}
		}
	})

	return auth.router.Handler()
}

// ClosePlugins calls Close for all registered plugins
func (auth *Auth) ClosePlugins() error {
	if auth.PluginRegistry == nil {
		return nil
	}

	auth.PluginRegistry.CloseAll()
	return nil
}
