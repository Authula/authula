package gobetterauth

import (
	"log/slog"
	"net/http"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/internal/auth"
	"github.com/GoBetterAuth/go-better-auth/internal/auth/storage"
	"github.com/GoBetterAuth/go-better-auth/internal/handlers"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// ---------------------------------
// INITIALISATION
// ---------------------------------

type Auth struct {
	Config      *domain.Config
	DB          *gorm.DB
	authService *auth.Service
}

func New(config *domain.Config, db *gorm.DB) *Auth {
	util.InitValidator()

	if db == nil {
		var err error
		switch config.Database.Provider {
		case "sqlite":
			db, err = gorm.Open(
				sqlite.Open(config.Database.ConnectionString),
				&gorm.Config{},
			)
		case "postgres":
			db, err = gorm.Open(
				postgres.Open(config.Database.ConnectionString),
				&gorm.Config{},
			)
		case "mysql":
			db, err = gorm.Open(
				mysql.Open(config.Database.ConnectionString),
				&gorm.Config{},
			)
		default:
			panic("unsupported database provider: " + config.Database.Provider)
		}
		if err != nil {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			logger.Error("failed to open database", slog.String("provider", config.Database.Provider), slog.String("connection_string", config.Database.ConnectionString), slog.Any("error", err))
			panic(err)
		}
	} else {
		// Validate the provided DB's provider
		switch db.Dialector.(type) {
		case *sqlite.Dialector:
			if config.Database.Provider != "sqlite" {
				panic("provided database provider does not match config: expected sqlite")
			}
		case *postgres.Dialector:
			if config.Database.Provider != "postgres" {
				panic("provided database provider does not match config: expected postgres")
			}
		case *mysql.Dialector:
			if config.Database.Provider != "mysql" {
				panic("provided database provider does not match config: expected mysql")
			}
		default:
			panic("unsupported database provider for provided DB instance")
		}
	}

	initStorage(config, db)

	return &Auth{
		Config:      config,
		DB:          db,
		authService: constructAuthService(config, db),
	}
}

func initStorage(config *domain.Config, db *gorm.DB) {
	if config.SecondaryStorage.Type == "" {
		if config.SecondaryStorage.Storage != nil {
			panic("secondary storage type of 'custom' must be specified")
		}

		// Default to in-memory secondary storage
		config.SecondaryStorage.Type = domain.SecondaryStorageTypeMemory
		config.SecondaryStorage.Storage = storage.NewMemorySecondaryStorage(config.SecondaryStorage.MemoryOptions)
	} else {
		switch config.SecondaryStorage.Type {
		case domain.SecondaryStorageTypeMemory:
			config.SecondaryStorage.Storage = storage.NewMemorySecondaryStorage(config.SecondaryStorage.MemoryOptions)
		case domain.SecondaryStorageTypeDatabase:
			config.SecondaryStorage.Storage = storage.NewDatabaseSecondaryStorage(db, config.SecondaryStorage.DatabaseOptions)
		case domain.SecondaryStorageTypeCustom:
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
		&domain.User{},
		&domain.Account{},
		&domain.Session{},
		&domain.Verification{},
		&domain.KeyValueStore{},
	}
	if err := auth.DB.AutoMigrate(models...); err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		logger.Error("failed to auto migrate database", slog.Any("error", err))
		panic(err)
	}
}

func (auth *Auth) DropMigrations() {
	// Drop domain tables
	models := []any{
		&domain.KeyValueStore{},
		&domain.Verification{},
		&domain.Session{},
		&domain.Account{},
		&domain.User{},
	}
	for _, model := range models {
		if err := auth.DB.Migrator().DropTable(model); err != nil {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			logger.Error("failed to drop table", slog.Any("model", model), slog.Any("error", err))
			panic(err)
		}
	}
}

// ---------------------------------
// MIDDLEWARES & HANDLERS
// ---------------------------------

func constructAuthService(config *domain.Config, db *gorm.DB) *auth.Service {
	userService := auth.NewUserService(config, db)
	accountService := auth.NewAccountService(config, db)
	sessionService := auth.NewSessionService(config, db)
	verificationService := auth.NewVerificationService(config, db)
	tokenService := auth.NewTokenService(config)
	rateLimitService := auth.NewRateLimitService(config)
	authService := auth.NewService(
		config,
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
		auth.authService,
		auth.Config.Session.CookieName,
	)
}

func (auth *Auth) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return middleware.OptionalAuthMiddleware(
		auth.authService,
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
	return middleware.RateLimitMiddleware(auth.authService.RateLimitService)
}

func (auth *Auth) EndpointHooksMiddleware() func(http.Handler) http.Handler {
	return middleware.EndpointHooksMiddleware(auth.Config, auth.authService)
}

func (auth *Auth) Handler() http.Handler {
	r := http.NewServeMux()

	// Handlers
	signIn := &handlers.SignInHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	signUp := &handlers.SignUpHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	signOut := &handlers.SignOutHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	sendEmailVerification := &handlers.SendEmailVerificationHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	verifyEmail := &handlers.VerifyEmailHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	resetPassword := &handlers.ResetPasswordHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	changePassword := &handlers.ChangePasswordHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	changeEmailRequest := &handlers.EmailChangeHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	me := &handlers.MeHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	oauth2Login := &handlers.OAuth2LoginHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}
	oauth2Callback := &handlers.OAuth2CallbackHandler{
		Config:      auth.Config,
		AuthService: auth.authService,
	}

	basePath := auth.Config.BasePath

	// Ensure basePath starts with "/" and does not end with "/"
	if basePath[0] != '/' {
		basePath = "/" + basePath
	}
	if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	// Routes
	r.Handle("POST "+basePath+"/sign-in/email", signIn.Handler())
	r.Handle("POST "+basePath+"/sign-up/email", signUp.Handler())
	r.Handle("POST "+basePath+"/email-verification", auth.AuthMiddleware()(auth.CSRFMiddleware()(sendEmailVerification.Handler())))
	r.Handle("GET "+basePath+"/verify-email", verifyEmail.Handler())
	r.Handle("POST "+basePath+"/sign-out", auth.AuthMiddleware()(auth.CSRFMiddleware()(signOut.Handler())))
	r.Handle("POST "+basePath+"/reset-password", resetPassword.Handler())
	r.Handle("POST "+basePath+"/change-password", changePassword.Handler())
	r.Handle("POST "+basePath+"/email-change", changeEmailRequest.Handler())
	r.Handle("GET "+basePath+"/me", auth.AuthMiddleware()(me.Handler()))
	r.Handle("GET "+basePath+"/oauth2/{provider}/login", oauth2Login.Handler())
	r.Handle("GET "+basePath+"/oauth2/{provider}/callback", oauth2Callback.Handler())

	var finalHandler http.Handler = r
	if auth.Config.RateLimit.Enabled {
		finalHandler = auth.RateLimitMiddleware()(finalHandler)
	}
	finalHandler = middleware.EndpointHooksMiddleware(auth.Config, auth.authService)(finalHandler)

	return finalHandler
}
