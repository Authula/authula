package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/config"
	corerepos "github.com/Authula/authula/core/repositories"
	coresecurity "github.com/Authula/authula/core/security"
	coreservices "github.com/Authula/authula/core/services"
	"github.com/Authula/authula/events"
	coreplugins "github.com/Authula/authula/internal/plugins"
	"github.com/Authula/authula/models"
	servicesinterfaces "github.com/Authula/authula/services"
)

type BaseTestFixture struct {
	T                   *testing.T
	DB                  *bun.DB
	Config              *models.Config
	Provider            string
	ServiceRegistry     models.ServiceRegistry
	UserService         servicesinterfaces.UserService
	AccountService      servicesinterfaces.AccountService
	SessionService      servicesinterfaces.SessionService
	VerificationService servicesinterfaces.VerificationService
	TokenService        servicesinterfaces.TokenService
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
	userRepo := corerepos.NewBunUserRepository(db)
	userService := coreservices.NewUserService(userRepo, nil)
	tokenRepo := corerepos.NewCryptoTokenRepository(cfg.Secret)
	accountRepo := corerepos.NewBunAccountRepository(db)
	accountService := coreservices.NewAccountService(cfg, accountRepo, tokenRepo, nil)
	sessionRepo := corerepos.NewBunSessionRepository(db)
	sessionService := coreservices.NewSessionService(sessionRepo, nil, nil)
	verificationRepo := corerepos.NewBunVerificationRepository(db)
	tokenSigner := coresecurity.NewHMACSigner(cfg.Secret)
	verificationService := coreservices.NewVerificationService(verificationRepo, tokenSigner, nil)
	tokenService := coreservices.NewTokenService(tokenRepo)

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
