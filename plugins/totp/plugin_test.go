package totp

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

type mockServiceRegistry struct {
	mock.Mock
}

func newMockServiceRegistry() *mockServiceRegistry {
	return &mockServiceRegistry{}
}

func (m *mockServiceRegistry) Register(name string, service any) {
	m.Called(name, service)
}

func (m *mockServiceRegistry) Get(name string) any {
	args := m.Called(name)
	return args.Get(0)
}

type mockPasswordService struct {
	mock.Mock
}

func (m *mockPasswordService) Verify(password, encoded string) bool {
	args := m.Called(password, encoded)
	return args.Bool(0)
}

func (m *mockPasswordService) Hash(password string) (string, error) {
	args := m.Called(password)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

type mockEventBus struct {
	mock.Mock
}

func (m *mockEventBus) Publish(ctx context.Context, event models.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockEventBus) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockEventBus) Subscribe(topic string, handler models.EventHandler) (models.SubscriptionID, error) {
	args := m.Called(topic, handler)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(models.SubscriptionID), args.Error(1)
}

func (m *mockEventBus) Unsubscribe(topic string, subscriptionID models.SubscriptionID) {
	m.Called(topic, subscriptionID)
}

type testLogger struct{}

func (testLogger) Debug(msg string, args ...any) {}
func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}

// mockEnableUseCase implements usecases.EnableUseCase for handler-level tests.
// ---------------------------------------------------------------------------
// Helper: build a fully-initialized plugin with mock services
// ---------------------------------------------------------------------------

func buildTestPlugin(t *testing.T) (*TOTPPlugin, *tests.MockUserService, *tests.MockTokenService, *tests.MockVerificationService) {
	t.Helper()

	userSvc := &tests.MockUserService{}
	accountSvc := &tests.MockAccountService{}
	sessionSvc := &tests.MockSessionService{}
	verifSvc := &tests.MockVerificationService{}
	tokenSvc := &tests.MockTokenService{}
	passwordSvc := &mockPasswordService{}
	eventBus := &mockEventBus{}

	reg := newMockServiceRegistry()
	reg.On("Get", models.ServiceUser.String()).Return(userSvc)
	reg.On("Get", models.ServiceAccount.String()).Return(accountSvc)
	reg.On("Get", models.ServiceSession.String()).Return(sessionSvc)
	reg.On("Get", models.ServiceVerification.String()).Return(verifSvc)
	reg.On("Get", models.ServiceToken.String()).Return(tokenSvc)
	reg.On("Get", models.ServicePassword.String()).Return(passwordSvc)

	passwordSvc.On("Verify", mock.Anything, mock.Anything).Return(true).Maybe()
	passwordSvc.On("Hash", mock.Anything).Return("hashed-password", nil).Maybe()

	eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
	eventBus.On("Close").Return(nil).Maybe()
	eventBus.On("Subscribe", mock.Anything, mock.Anything).Return(models.SubscriptionID(0), nil).Maybe()
	eventBus.On("Unsubscribe", mock.Anything, mock.Anything).Return().Maybe()

	plugin := New(types.TOTPPluginConfig{})

	pluginCtx := &models.PluginContext{
		DB:              nil, // not used in hook/handler tests
		Logger:          &tests.MockLogger{},
		EventBus:        eventBus,
		ServiceRegistry: reg,
		GetConfig: func() *models.Config {
			return &models.Config{
				Session: models.SessionConfig{
					ExpiresIn: 24 * time.Hour,
				},
			}
		},
	}

	if err := plugin.Init(pluginCtx); err != nil {
		t.Fatalf("plugin.Init failed: %v", err)
	}

	return plugin, userSvc, tokenSvc, verifSvc
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPluginInit(t *testing.T) {
	plugin, _, _, _ := buildTestPlugin(t)

	routes := plugin.Routes()
	if len(routes) != 7 {
		t.Fatalf("expected 7 routes, got %d", len(routes))
	}

	hooks := plugin.Hooks()
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}

	if hooks[0].Stage != models.HookAfter {
		t.Errorf("expected HookAfter stage, got %v", hooks[0].Stage)
	}
}

// Note: TestAfterHookPassesThroughWithout2FA and TestAfterHookIntercepts2FAUser
// were removed because the hook now uses repo.IsEnabled() which requires a real
// database connection. These scenarios should be covered by integration tests.

func newHookTestDB(t *testing.T) *bun.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	migrator, err := migrations.NewMigrator(db, testLogger{})
	require.NoError(t, err)

	coreSet, err := migrations.CoreMigrationSet("sqlite")
	require.NoError(t, err)
	totpSet := MigrationSet("sqlite")

	err = migrator.Migrate(ctx, []migrations.MigrationSet{coreSet, totpSet})
	require.NoError(t, err)

	return db
}

func TestInterceptHook_DoesNotBypassWithOtherUsersTrustedToken(t *testing.T) {
	plugin, _, tokenSvc, verifSvc := buildTestPlugin(t)
	db := newHookTestDB(t)
	repo := repository.NewTOTPRepository(db)
	plugin.totpRepo = repo

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-1", "User One", "user1@example.com")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-2", "User Two", "user2@example.com")
	require.NoError(t, err)

	_, err = repo.Create(ctx, "user-1", "enc-secret", "[]")
	require.NoError(t, err)
	require.NoError(t, repo.SetEnabled(ctx, "user-1", true))

	_, err = repo.CreateTrustedDevice(ctx, "user-2", "hash-trusted", "ua", time.Now().UTC().Add(24*time.Hour))
	require.NoError(t, err)

	tokenSvc.On("Hash", "trusted-cookie-token").Return("hash-trusted").Once()
	tokenSvc.On("Generate").Return("raw-pending", nil).Once()
	tokenSvc.On("Hash", "raw-pending").Return("hash-pending").Once()
	verifSvc.On(
		"Create",
		mock.Anything,
		"user-1",
		"hash-pending",
		models.TypeTOTPPendingAuth,
		"user-1",
		plugin.pluginConfig.PendingTokenExpiry,
	).Return(&models.Verification{ID: "verif-1"}, nil).Once()

	req := httptest.NewRequest(http.MethodPost, "/sign-in", nil)
	req.AddCookie(&http.Cookie{Name: constants.CookieTOTPTrusted, Value: "trusted-cookie-token"})
	w := httptest.NewRecorder()

	userID := "user-1"
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values: map[string]any{
			models.ContextAuthSuccess.String(): true,
		},
		UserID: &userID,
	}

	err = plugin.interceptSignInHook(reqCtx)
	require.NoError(t, err)

	_, authSuccessExists := reqCtx.Values[models.ContextAuthSuccess.String()]
	assert.False(t, authSuccessExists)
	assert.Equal(t, http.StatusOK, reqCtx.ResponseStatus)

	cookies := w.Result().Cookies()
	var pendingCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == constants.CookieTOTPPending {
			pendingCookie = c
			break
		}
	}
	require.NotNil(t, pendingCookie)
	assert.Equal(t, "raw-pending", pendingCookie.Value)

	tokenSvc.AssertExpectations(t)
	verifSvc.AssertExpectations(t)
}

func TestInterceptHook_BypassesWithMatchingTrustedDevice(t *testing.T) {
	plugin, _, tokenSvc, verifSvc := buildTestPlugin(t)
	db := newHookTestDB(t)
	repo := repository.NewTOTPRepository(db)
	plugin.totpRepo = repo

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-1", "User One", "user1@example.com")
	require.NoError(t, err)

	_, err = repo.Create(ctx, "user-1", "enc-secret", "[]")
	require.NoError(t, err)
	require.NoError(t, repo.SetEnabled(ctx, "user-1", true))

	_, err = repo.CreateTrustedDevice(ctx, "user-1", "hash-trusted", "ua", time.Now().UTC().Add(24*time.Hour))
	require.NoError(t, err)

	tokenSvc.On("Hash", "trusted-cookie-token").Return("hash-trusted").Once()

	req := httptest.NewRequest(http.MethodPost, "/sign-in", nil)
	req.AddCookie(&http.Cookie{Name: constants.CookieTOTPTrusted, Value: "trusted-cookie-token"})
	w := httptest.NewRecorder()

	userID := "user-1"
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values: map[string]any{
			models.ContextAuthSuccess.String(): true,
		},
		UserID: &userID,
	}

	err = plugin.interceptSignInHook(reqCtx)
	require.NoError(t, err)

	authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
	assert.True(t, ok)
	assert.True(t, authSuccess)
	assert.False(t, reqCtx.ResponseReady)
	assert.Empty(t, w.Result().Cookies())

	tokenSvc.AssertExpectations(t)
	verifSvc.AssertExpectations(t)
}
