package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/config"
	"github.com/GoBetterAuth/go-better-auth/v2/events"
	coreplugins "github.com/GoBetterAuth/go-better-auth/v2/internal/plugins"
	coreinternalrepos "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	coreinternalsecurity "github.com/GoBetterAuth/go-better-auth/v2/internal/security"
	coreinternalservices "github.com/GoBetterAuth/go-better-auth/v2/internal/services"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	coreservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type BaseTestFixture struct {
	T                   *testing.T
	DB                  *bun.DB
	Config              *models.Config
	Provider            string
	ServiceRegistry     models.ServiceRegistry
	UserService         coreservices.UserService
	AccountService      coreservices.AccountService
	SessionService      coreservices.SessionService
	VerificationService coreservices.VerificationService
	TokenService        coreservices.TokenService
	idAliases           map[string]string
}

func NewBaseTestFixture(t *testing.T, options ...config.ConfigOption) *BaseTestFixture {
	t.Helper()

	db, provider := NewIntegrationTestDBFromEnv(t)

	defaultOptions := []config.ConfigOption{
		config.WithBasePath("/auth"),
		config.WithSecret("integration-test-secret"),
		config.WithDatabase(models.DatabaseConfig{Provider: provider}),
		config.WithEventBus(models.EventBusConfig{Provider: events.ProviderGoChannel}),
	}

	cfg := config.NewConfig(append(defaultOptions, options...)...)

	serviceRegistry := coreplugins.NewServiceRegistry()
	userRepo := coreinternalrepos.NewBunUserRepository(db)
	userService := coreinternalservices.NewUserService(userRepo, nil)
	tokenRepo := coreinternalrepos.NewCryptoTokenRepository(cfg.Secret)
	accountRepo := coreinternalrepos.NewBunAccountRepository(db)
	accountService := coreinternalservices.NewAccountService(cfg, accountRepo, tokenRepo, nil)
	sessionRepo := coreinternalrepos.NewBunSessionRepository(db)
	sessionService := coreinternalservices.NewSessionService(sessionRepo, nil, nil)
	verificationRepo := coreinternalrepos.NewBunVerificationRepository(db)
	tokenSigner := coreinternalsecurity.NewHMACSigner(cfg.Secret)
	verificationService := coreinternalservices.NewVerificationService(verificationRepo, tokenSigner, nil)
	tokenService := coreinternalservices.NewTokenService(tokenRepo)

	serviceRegistry.Register(models.ServiceUser.String(), userService)
	serviceRegistry.Register(models.ServiceAccount.String(), accountService)
	serviceRegistry.Register(models.ServiceSession.String(), sessionService)
	serviceRegistry.Register(models.ServiceVerification.String(), verificationService)
	serviceRegistry.Register(models.ServiceToken.String(), tokenService)

	return &BaseTestFixture{
		T:                   t,
		DB:                  db,
		Config:              cfg,
		Provider:            provider,
		ServiceRegistry:     serviceRegistry,
		UserService:         userService,
		AccountService:      accountService,
		SessionService:      sessionService,
		VerificationService: verificationService,
		TokenService:        tokenService,
		idAliases:           make(map[string]string),
	}
}

func (f *BaseTestFixture) ResolveID(id string) string {
	f.T.Helper()
	if mapped, ok := f.idAliases[id]; ok {
		return mapped
	}

	if _, err := uuid.Parse(id); err == nil {
		return id
	}

	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(id)).String()
}

func (f *BaseTestFixture) SeedUser(id, email string) string {
	f.T.Helper()
	created, err := f.UserService.Create(context.Background(), "Integration User", email, false, nil, nil)
	if err != nil {
		f.T.Fatalf("failed to seed user: %v", err)
	}

	f.idAliases[id] = created.ID
	return created.ID
}

func (f *BaseTestFixture) SeedSession(sessionID, userID string) string {
	f.T.Helper()
	resolvedUserID := f.ResolveID(userID)
	created, err := f.SessionService.Create(
		context.Background(),
		resolvedUserID,
		"token-"+f.ResolveID(sessionID),
		nil,
		nil,
		30*time.Minute,
	)
	if err != nil {
		f.T.Fatalf("failed to seed session: %v", err)
	}

	f.idAliases[sessionID] = created.ID
	return created.ID
}
