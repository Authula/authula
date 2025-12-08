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
	"github.com/GoBetterAuth/go-better-auth/internal/handlers"
	"github.com/GoBetterAuth/go-better-auth/internal/middleware"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// Auth is the main struct exposing the unified handler
type Auth struct {
	Config *domain.Config
	DB     *gorm.DB
}

// New initializes the Auth library with GORM
func New(config *domain.Config, db *gorm.DB) *Auth {
	util.InitValidator()

	var dbToUse *gorm.DB
	if db == nil {
		var err error
		switch config.Database.Provider {
		case "sqlite":
			dbToUse, err = gorm.Open(
				sqlite.Open(config.Database.ConnectionString),
				&gorm.Config{},
			)
		case "postgres":
			dbToUse, err = gorm.Open(
				postgres.Open(config.Database.ConnectionString),
				&gorm.Config{},
			)
		case "mysql":
			dbToUse, err = gorm.Open(
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
		dbToUse = db
	}

	return &Auth{
		Config: config,
		DB:     dbToUse,
	}
}

func (auth *Auth) RunMigrations() {
	// Auto migrate domain models
	if err := auth.DB.AutoMigrate(&domain.User{}, &domain.Account{}, &domain.Session{}, &domain.Verification{}); err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		logger.Error("failed to auto migrate database", slog.Any("error", err))
		panic(err)
	}
}

func (auth *Auth) DropMigrations() {
	// Drop domain tables
	models := []any{
		&domain.User{},
		&domain.Account{},
		&domain.Session{},
		&domain.Verification{},
	}
	for _, model := range models {
		if err := auth.DB.Migrator().DropTable(model); err != nil {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			logger.Error("failed to drop table", slog.Any("model", model), slog.Any("error", err))
			panic(err)
		}
	}
}

func (auth *Auth) Handler() http.Handler {
	r := http.NewServeMux()

	authService := constructAuthService(auth.Config, auth.DB)

	// Handlers
	signIn := &handlers.SignInHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	signUp := &handlers.SignUpHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	signOut := &handlers.SignOutHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	sendEmailVerification := &handlers.SendEmailVerificationHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	verifyEmail := &handlers.VerifyEmailHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	resetPassword := &handlers.ResetPasswordHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	changePassword := &handlers.ChangePasswordHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	changeEmailRequest := &handlers.EmailChangeHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	me := &handlers.MeHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	oauth2Login := &handlers.OAuth2LoginHandler{
		Config:      auth.Config,
		AuthService: authService,
	}
	oauth2Callback := &handlers.OAuth2CallbackHandler{
		Config:      auth.Config,
		AuthService: authService,
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
	r.Handle("GET "+basePath+"/me", auth.AuthMiddleware()(auth.CSRFMiddleware()(me.Handler())))
	r.Handle("GET "+basePath+"/oauth2/{provider}/login", oauth2Login.Handler())
	r.Handle("GET "+basePath+"/oauth2/{provider}/callback", oauth2Callback.Handler())

	return middleware.EndpointHooksMiddleware(auth.Config, authService)(r)
}

// Export middleware
func (auth *Auth) AuthMiddleware() func(http.Handler) http.Handler {
	return middleware.AuthMiddleware(
		constructAuthService(auth.Config, auth.DB),
		auth.Config.Session.CookieName,
	)
}

func (auth *Auth) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return middleware.OptionalAuthMiddleware(
		constructAuthService(auth.Config, auth.DB),
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

func constructAuthService(config *domain.Config, db *gorm.DB) *auth.Service {
	userService := auth.NewUserService(config, db)
	accountService := auth.NewAccountService(config, db)
	sessionService := auth.NewSessionService(config, db)
	verificationService := auth.NewVerificationService(config, db)
	tokenService := auth.NewTokenService(config)
	authService := auth.NewService(
		config,
		userService,
		accountService,
		sessionService,
		verificationService,
		tokenService,
	)

	return authService
}
