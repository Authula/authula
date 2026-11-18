package totp

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"github.com/Authula/authula/internal/sqlitedsn"
	"github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/constants"
	"github.com/Authula/authula/plugins/totp/repository"
	"github.com/Authula/authula/plugins/totp/types"
)

// stubServices holds the six core services the TOTP plugin requires.
type stubServices struct {
	user         *tests.MockUserService
	account      *tests.MockAccountService
	session      *tests.MockSessionService
	verification *tests.MockVerificationService
	token        *tests.MockTokenService
	password     *tests.MockPasswordService
}

func newStubServices() stubServices {
	return stubServices{
		user:         &tests.MockUserService{},
		account:      &tests.MockAccountService{},
		session:      &tests.MockSessionService{},
		verification: &tests.MockVerificationService{},
		token:        &tests.MockTokenService{},
		password:     &tests.MockPasswordService{},
	}
}

// registry returns a registry that serves every stub except the named service.
func (s stubServices) registry(omit models.ServiceID) *tests.MockServiceRegistry {
	registry := &tests.MockServiceRegistry{}
	entries := []struct {
		id      models.ServiceID
		service any
	}{
		{models.ServiceUser, s.user},
		{models.ServiceAccount, s.account},
		{models.ServiceSession, s.session},
		{models.ServiceVerification, s.verification},
		{models.ServiceToken, s.token},
		{models.ServicePassword, s.password},
	}

	for _, entry := range entries {
		if entry.id == omit {
			registry.On("Get", entry.id.String()).Return(nil)
			continue
		}
		registry.On("Get", entry.id.String()).Return(entry.service)
	}

	return registry
}

func newPluginContext(registry models.ServiceRegistry, logger models.Logger, cfg *models.Config) *models.PluginContext {
	return &models.PluginContext{
		Logger:          logger,
		EventBus:        &tests.MockEventBus{},
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return cfg },
	}
}

func buildTestPlugin(t *testing.T) (*TOTPPlugin, *tests.MockUserService, *tests.MockTokenService, *tests.MockVerificationService) {
	t.Helper()

	userSvc := &tests.MockUserService{}
	accountSvc := &tests.MockAccountService{}
	sessionSvc := &tests.MockSessionService{}
	verifSvc := &tests.MockVerificationService{}
	tokenSvc := &tests.MockTokenService{}
	passwordSvc := &tests.MockPasswordService{}
	eventBus := &tests.MockEventBus{}

	serviceRegistry := &tests.MockServiceRegistry{}
	serviceRegistry.On("Get", models.ServiceUser.String()).Return(userSvc)
	serviceRegistry.On("Get", models.ServiceAccount.String()).Return(accountSvc)
	serviceRegistry.On("Get", models.ServiceSession.String()).Return(sessionSvc)
	serviceRegistry.On("Get", models.ServiceVerification.String()).Return(verifSvc)
	serviceRegistry.On("Get", models.ServiceToken.String()).Return(tokenSvc)
	serviceRegistry.On("Get", models.ServicePassword.String()).Return(passwordSvc)

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
		ServiceRegistry: serviceRegistry,
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

func newHookTestDB(t *testing.T) *bun.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	migrator, err := migrations.NewMigrator(db, &tests.MockLogger{})
	require.NoError(t, err)

	coreSet, err := migrations.CoreMigrationSet("sqlite")
	require.NoError(t, err)
	totpSet := MigrationSet("sqlite")

	err = migrator.Migrate(ctx, []migrations.MigrationSet{coreSet, totpSet})
	require.NoError(t, err)

	return db
}

func findPendingCookieValue(w *httptest.ResponseRecorder) (string, bool) {
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == constants.CookieTOTPPending {
			return cookie.Value, true
		}
	}

	return "", false
}

func TestInterceptHook(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(t *testing.T, plugin *TOTPPlugin, db *bun.DB, totpRepo *repository.TOTPRepository, tokenSvc *tests.MockTokenService, verifSvc *tests.MockVerificationService)
		assert  func(t *testing.T, reqCtx *models.RequestContext, w *httptest.ResponseRecorder)
	}{
		{
			name: "does not bypass with another user's trusted device",
			arrange: func(t *testing.T, plugin *TOTPPlugin, db *bun.DB, totpRepo *repository.TOTPRepository, tokenSvc *tests.MockTokenService, verifSvc *tests.MockVerificationService) {
				t.Helper()

				ctx := context.Background()
				_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-1", "User One", "user1@example.com")
				require.NoError(t, err)
				_, err = db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-2", "User Two", "user2@example.com")
				require.NoError(t, err)

				plugin.totpRepo = totpRepo

				_, err = totpRepo.Create(ctx, "user-1", "enc-secret", "[]")
				require.NoError(t, err)
				require.NoError(t, totpRepo.SetEnabled(ctx, "user-1", true))

				_, err = totpRepo.CreateTrustedDevice(ctx, "user-2", "hash-trusted", "ua", time.Now().UTC().Add(24*time.Hour))
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
			},
			assert: func(t *testing.T, reqCtx *models.RequestContext, w *httptest.ResponseRecorder) {
				t.Helper()

				_, authSuccessExists := reqCtx.Values[models.ContextAuthSuccess.String()]
				assert.False(t, authSuccessExists)
				assert.Equal(t, http.StatusOK, reqCtx.ResponseStatus)

				pendingCookieValue, ok := findPendingCookieValue(w)
				require.True(t, ok)
				assert.Equal(t, "raw-pending", pendingCookieValue)
			},
		},
		{
			name: "bypasses with a matching trusted device",
			arrange: func(t *testing.T, plugin *TOTPPlugin, db *bun.DB, totpRepo *repository.TOTPRepository, tokenSvc *tests.MockTokenService, verifSvc *tests.MockVerificationService) {
				t.Helper()

				ctx := context.Background()
				_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, "user-1", "User One", "user1@example.com")
				require.NoError(t, err)

				plugin.totpRepo = totpRepo

				_, err = totpRepo.Create(ctx, "user-1", "enc-secret", "[]")
				require.NoError(t, err)
				require.NoError(t, totpRepo.SetEnabled(ctx, "user-1", true))

				_, err = totpRepo.CreateTrustedDevice(ctx, "user-1", "hash-trusted", "ua", time.Now().UTC().Add(24*time.Hour))
				require.NoError(t, err)

				tokenSvc.On("Hash", "trusted-cookie-token").Return("hash-trusted").Once()
			},
			assert: func(t *testing.T, reqCtx *models.RequestContext, w *httptest.ResponseRecorder) {
				t.Helper()

				authSuccess, ok := reqCtx.Values[models.ContextAuthSuccess.String()].(bool)
				assert.True(t, ok)
				assert.True(t, authSuccess)
				assert.False(t, reqCtx.ResponseReady)
				assert.Empty(t, w.Result().Cookies())
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plugin, _, tokenSvc, verifSvc := buildTestPlugin(t)
			db := newHookTestDB(t)
			totpRepo := repository.NewTOTPRepository(db)

			tc.arrange(t, plugin, db, totpRepo, tokenSvc, verifSvc)

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
				Actor: &models.Actor{ID: userID, Type: models.ActorUser},
			}

			err := plugin.interceptSignInHook(reqCtx)
			require.NoError(t, err)

			tc.assert(t, reqCtx, w)

			tokenSvc.AssertExpectations(t)
			verifSvc.AssertExpectations(t)
		})
	}
}

func TestTOTPPlugin_Config(t *testing.T) {
	t.Parallel()

	plugin := New(types.TOTPPluginConfig{BackupCodeCount: 4, SameSite: "strict"})

	cfg, ok := plugin.Config().(*types.TOTPPluginConfig)
	require.True(t, ok)
	assert.Equal(t, 4, cfg.BackupCodeCount)
	assert.Equal(t, "strict", cfg.SameSite)
	assert.Equal(t, 30*24*time.Hour, cfg.TrustedDeviceDuration)
	assert.Equal(t, 5*time.Minute, cfg.PendingTokenExpiry)
}

func TestTOTPPlugin_InitMissingService(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		omit    models.ServiceID
		wantErr string
	}{
		{name: "user service", omit: models.ServiceUser, wantErr: "user service not available"},
		{name: "account service", omit: models.ServiceAccount, wantErr: "account service not available"},
		{name: "session service", omit: models.ServiceSession, wantErr: "session service not available"},
		{name: "verification service", omit: models.ServiceVerification, wantErr: "verification service not available"},
		{name: "token service", omit: models.ServiceToken, wantErr: "token service not available"},
		{name: "password service", omit: models.ServicePassword, wantErr: "password service not available"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := newStubServices().registry(tt.omit)
			plugin := New(types.TOTPPluginConfig{})

			err := plugin.Init(newPluginContext(registry, &tests.MockLogger{}, &models.Config{}))

			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, plugin.Api)
		})
	}
}

func TestTOTPPlugin_InitWarnsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	logger := &tests.LevelRecordingLogger{}
	cfg := &models.Config{
		Plugins: models.PluginsConfig{
			models.PluginTOTP.String(): map[string]any{"pending_token_expiry": "not-a-duration"},
		},
	}
	plugin := New(types.TOTPPluginConfig{})

	err := plugin.Init(newPluginContext(newStubServices().registry(""), logger, cfg))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, plugin.Close()) })

	warnings := 0
	for _, record := range logger.Records() {
		if record.Level == "warn" && record.Message == "failed to load totp plugin config, using defaults" {
			warnings++
		}
	}
	assert.Equal(t, 1, warnings)
	assert.NotNil(t, plugin.Api)
}

func TestTOTPPlugin_InitWiresServices(t *testing.T) {
	t.Parallel()

	stubs := newStubServices()
	plugin := New(types.TOTPPluginConfig{BackupCodeCount: 7})

	err := plugin.Init(newPluginContext(stubs.registry(""), &tests.MockLogger{}, &models.Config{
		Session: models.SessionConfig{Domain: "example.com"},
	}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, plugin.Close()) })

	assert.Equal(t, stubs.user, plugin.userService)
	assert.Equal(t, stubs.account, plugin.accountService)
	assert.Equal(t, stubs.session, plugin.sessionService)
	assert.Equal(t, stubs.verification, plugin.verificationService)
	assert.Equal(t, stubs.token, plugin.tokenService)
	assert.Equal(t, stubs.password, plugin.passwordService)

	require.NotNil(t, plugin.totpService)
	assert.Equal(t, 6, plugin.totpService.Digits)
	assert.Equal(t, 30, plugin.totpService.PeriodSeconds)

	require.NotNil(t, plugin.backupCodeService)
	assert.Equal(t, 7, plugin.backupCodeService.Count)
	assert.Equal(t, stubs.password, plugin.backupCodeService.PasswordService)

	require.NotNil(t, plugin.totpRepo)
	require.NotNil(t, plugin.cleanupSystem)

	require.NotNil(t, plugin.Api)
	useCases := plugin.Api.UseCases()
	require.NotNil(t, useCases)
	assert.NotNil(t, useCases.Enable)
	assert.NotNil(t, useCases.Disable)
	assert.NotNil(t, useCases.GetTOTPURI)
	assert.NotNil(t, useCases.VerifyTOTP)
	assert.NotNil(t, useCases.GenerateBackupCodes)
	assert.NotNil(t, useCases.VerifyBackupCode)
}

func TestTOTPPlugin_Close(t *testing.T) {
	t.Parallel()

	t.Run("disabled auto cleanup is repeatable", func(t *testing.T) {
		t.Parallel()

		plugin := New(types.TOTPPluginConfig{})
		require.NoError(t, plugin.Init(newPluginContext(newStubServices().registry(""), &tests.MockLogger{}, &models.Config{})))

		require.NoError(t, plugin.Close())
		require.NoError(t, plugin.Close())
	})

	t.Run("enabled auto cleanup stops the loop", func(t *testing.T) {
		t.Parallel()

		plugin := New(types.TOTPPluginConfig{
			TrustedDevicesAutoCleanup:     true,
			TrustedDevicesCleanupInterval: time.Hour,
		})
		require.NoError(t, plugin.Init(newPluginContext(newStubServices().registry(""), &tests.MockLogger{}, &models.Config{})))

		// Close blocks until the cleanup goroutine has returned.
		require.NoError(t, plugin.Close())
	})
}

func TestTOTPPlugin_Routes(t *testing.T) {
	t.Parallel()

	plugin, _, _, _ := buildTestPlugin(t)

	routes := plugin.Routes()

	expected := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/totp/enable"},
		{method: http.MethodPost, path: "/totp/disable"},
		{method: http.MethodGet, path: "/totp/get-uri"},
		{method: http.MethodPost, path: "/totp/verify"},
		{method: http.MethodPost, path: "/totp/verify-backup-code"},
		{method: http.MethodPost, path: "/totp/generate-backup-codes"},
	}

	require.Len(t, routes, len(expected))
	for i, tt := range expected {
		assert.Equal(t, tt.method, routes[i].Method)
		assert.Equal(t, tt.path, routes[i].Path)
		assert.NotEmpty(t, routes[i].Middleware)
		assert.NotNil(t, routes[i].Handler)
	}
}

func TestTOTPPlugin_Hooks(t *testing.T) {
	t.Parallel()

	plugin := New(types.TOTPPluginConfig{})
	hooks := plugin.Hooks()

	require.Len(t, hooks, 1)
	assert.Equal(t, models.HookAfter, hooks[0].Stage)
	assert.Equal(t, HookIDTOTPIntercept.String(), hooks[0].PluginID)
	assert.Equal(t, 5, hooks[0].Order)
	require.NotNil(t, hooks[0].Matcher)
	require.NotNil(t, hooks[0].Handler)
}

func TestTOTPPlugin_Migrations(t *testing.T) {
	t.Parallel()

	plugin := New(types.TOTPPluginConfig{})

	tests := []struct {
		name     string
		provider string
		want     int
	}{
		{name: "sqlite", provider: "sqlite", want: 1},
		{name: "postgres", provider: "postgres", want: 1},
		{name: "mysql", provider: "mysql", want: 1},
		{name: "unknown provider", provider: "cockroach", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Len(t, plugin.Migrations(tt.provider), tt.want)
		})
	}
}

func TestTOTPPlugin_DependsOn(t *testing.T) {
	t.Parallel()

	assert.Nil(t, New(types.TOTPPluginConfig{}).DependsOn())
}

func TestTOTPHookID_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "totp.intercept", HookIDTOTPIntercept.String())
}
