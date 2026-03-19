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

	sqlDB, err := sql.Open("sqlite3", ":memory:")
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
				UserID: &userID,
			}

			err := plugin.interceptSignInHook(reqCtx)
			require.NoError(t, err)

			tc.assert(t, reqCtx, w)

			tokenSvc.AssertExpectations(t)
			verifSvc.AssertExpectations(t)
		})
	}
}
