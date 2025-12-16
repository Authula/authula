package gobetterauth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/events"
	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/handlers"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/plugins"
	"github.com/GoBetterAuth/go-better-auth/internal/services"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/storage"
)

// ---------------------------------
// INITIALISATION
// ---------------------------------

type Auth struct {
	Config         *models.Config
	mux            *http.ServeMux
	service        *auth.Service
	Api            *models.Api
	customRoutes   []models.CustomRoute
	pluginRegistry *plugins.PluginRegistry
}

func New(config *models.Config) *Auth {
	util.InitValidator()
	initStorage(config)
	mux := http.NewServeMux()

	var eventBus models.EventBus
	if config.EventBus.Enabled {
		eventBus = events.NewEventBus(config, config.EventBus.PubSub)
	}

	auth := &Auth{
		Config:       config,
		mux:          mux,
		customRoutes: []models.CustomRoute{},
	}

	pluginMiddleware := &models.PluginMiddleware{
		Auth:          auth.AuthMiddleware,
		OptionalAuth:  auth.OptionalAuthMiddleware,
		CorsAuth:      auth.CorsAuthMiddleware,
		CSRF:          auth.CSRFMiddleware,
		RateLimit:     auth.RateLimitMiddleware,
		EndpointHooks: auth.EndpointHooksMiddleware,
	}

	pluginRateLimits := []models.PluginRateLimit{}
	for _, p := range config.Plugins.Plugins {
		if rateLimit := p.RateLimit(); rateLimit != nil && rateLimit.Enabled {
			pluginRateLimits = append(pluginRateLimits, *rateLimit)
		}
	}

	authService := constructAuthService(config, eventBus, pluginRateLimits)

	api := &models.Api{
		Users:         authService.UserService,
		Accounts:      authService.AccountService,
		Sessions:      authService.SessionService,
		Verifications: authService.VerificationService,
		Tokens:        authService.TokenService,
	}

	pluginRegistry := plugins.NewPluginRegistry(config, api, eventBus, pluginMiddleware)
	for _, p := range config.Plugins.Plugins {
		pluginRegistry.Register(p)
	}
	_ = pluginRegistry.InitAll()

	auth.service = authService
	auth.Api = api
	auth.pluginRegistry = pluginRegistry

	return auth
}

func initStorage(config *models.Config) {
	if config.SecondaryStorage.Type == "" {
		if config.SecondaryStorage.Storage != nil {
			panic("secondary storage type of 'custom' must be specified")
		}

		// Default to in-memory secondary storage
		config.SecondaryStorage.Type = models.SecondaryStorageTypeMemory
		config.SecondaryStorage.Storage = storage.NewMemorySecondaryStorage(config.SecondaryStorage.MemoryOptions)
	} else {
		switch config.SecondaryStorage.Type {
		case models.SecondaryStorageTypeMemory:
			config.SecondaryStorage.Storage = storage.NewMemorySecondaryStorage(config.SecondaryStorage.MemoryOptions)
		case models.SecondaryStorageTypeDatabase:
			config.SecondaryStorage.Storage = storage.NewDatabaseSecondaryStorage(config.DB, config.SecondaryStorage.DatabaseOptions)
		case models.SecondaryStorageTypeCustom:
			// Valid, do nothing
		default:
			panic("unsupported secondary storage type: " + config.SecondaryStorage.Type)
		}
	}
}

// ---------------------------------
// MIGRATIONS
// ---------------------------------

func (auth *Auth) RunMigrations() {
	models := []any{
		&models.User{},
		&models.Account{},
		&models.Session{},
		&models.Verification{},
		&models.KeyValueStore{},
	}
	if err := auth.Config.DB.AutoMigrate(models...); err != nil {
		slog.Error("failed to auto migrate database", slog.Any("error", err))
		panic(err)
	}

	if err := auth.pluginRegistry.RunMigrations(); err != nil {
		slog.Error("failed to run plugin migrations", slog.Any("error", err))
		panic(err)
	}
}

func (auth *Auth) DropMigrations() {
	models := []any{
		&models.KeyValueStore{},
		&models.Verification{},
		&models.Session{},
		&models.Account{},
		&models.User{},
	}
	for _, model := range models {
		if err := auth.Config.DB.Migrator().DropTable(model); err != nil {
			slog.Error("failed to drop table", slog.Any("model", model), slog.Any("error", err))
			panic(err)
		}
	}
}

// ---------------------------------
// MIDDLEWARES & HANDLERS
// ---------------------------------

func constructAuthService(config *models.Config, eventBus models.EventBus, pluginRateLimits []models.PluginRateLimit) *auth.Service {
	userService := services.NewUserServiceImpl(config, config.DB)
	accountService := services.NewAccountServiceImpl(config, config.DB)
	sessionService := services.NewSessionServiceImpl(config, config.DB)
	verificationService := services.NewVerificationServiceImpl(config, config.DB)
	tokenService := services.NewTokenServiceImpl(config)
	rateLimitService := services.NewRateLimitServiceImpl(config, pluginRateLimits)

	authService := auth.NewService(
		config,
		eventBus,
		userService,
		accountService,
		sessionService,
		verificationService,
		tokenService,
		rateLimitService,
	)

	return authService
}

func (auth *Auth) AuthMiddleware() func(http.Handler) http.Handler {
	return middleware.AuthMiddleware(
		auth.service,
		auth.Config.Session.CookieName,
	)
}

func (auth *Auth) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return middleware.OptionalAuthMiddleware(
		auth.service,
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
	return middleware.RateLimitMiddleware(auth.service.RateLimitService)
}

func (auth *Auth) EndpointHooksMiddleware() func(http.Handler) http.Handler {
	return middleware.EndpointHooksMiddleware(auth.Config, auth.service)
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
	auth.customRoutes = append(auth.customRoutes, route)
}

// Handler sets up all routes and returns the final http.Handler
func (auth *Auth) Handler() http.Handler {
	signIn := &handlers.SignInHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	signUp := &handlers.SignUpHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	signOut := &handlers.SignOutHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	sendEmailVerification := &handlers.SendEmailVerificationHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	verifyEmail := &handlers.VerifyEmailHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	resetPassword := &handlers.ResetPasswordHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	changePassword := &handlers.ChangePasswordHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	changeEmailRequest := &handlers.EmailChangeHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	me := &handlers.MeHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	oauth2Login := &handlers.OAuth2LoginHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}
	oauth2Callback := &handlers.OAuth2CallbackHandler{
		Config:      auth.Config,
		AuthService: auth.service,
	}

	basePath := auth.Config.BasePath

	// Ensure basePath starts with "/" and does not end with "/"
	if basePath[0] != '/' {
		basePath = "/" + basePath
	}
	if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	// Base routes
	auth.mux.Handle("POST "+basePath+"/sign-in/email", signIn.Handler())
	auth.mux.Handle("POST "+basePath+"/sign-up/email", signUp.Handler())
	auth.mux.Handle("POST "+basePath+"/email-verification", auth.AuthMiddleware()(auth.CSRFMiddleware()(sendEmailVerification.Handler())))
	auth.mux.Handle("GET "+basePath+"/verify-email", verifyEmail.Handler())
	auth.mux.Handle("POST "+basePath+"/sign-out", auth.AuthMiddleware()(auth.CSRFMiddleware()(signOut.Handler())))
	auth.mux.Handle("POST "+basePath+"/reset-password", resetPassword.Handler())
	auth.mux.Handle("POST "+basePath+"/change-password", changePassword.Handler())
	auth.mux.Handle("POST "+basePath+"/email-change", changeEmailRequest.Handler())
	auth.mux.Handle("GET "+basePath+"/me", auth.AuthMiddleware()(me.Handler()))
	auth.mux.Handle("GET "+basePath+"/oauth2/{provider}/login", oauth2Login.Handler())
	auth.mux.Handle("GET "+basePath+"/oauth2/{provider}/callback", oauth2Callback.Handler())

	auth.registerCustomRoutes(basePath)
	auth.registerPluginRoutes(basePath)

	var finalHandler http.Handler = auth.mux
	finalHandler = middleware.EndpointHooksMiddleware(auth.Config, auth.service)(finalHandler)
	if auth.Config.RateLimit.Enabled {
		finalHandler = auth.RateLimitMiddleware()(finalHandler)
	}

	return finalHandler
}

func (auth *Auth) registerCustomRoutes(basePath string) {
	if len(auth.customRoutes) > 0 {
		for _, customRoute := range auth.customRoutes {
			path := fmt.Sprintf("%s%s", basePath, customRoute.Path)
			auth.mux.Handle(fmt.Sprintf("%s %s", customRoute.Method, path), customRoute.Handler(auth.Config))
		}
	}
}

// RegisterPluginRoutes registers routes from plugins
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
