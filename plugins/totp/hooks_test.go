package totp

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/constants"
	"github.com/Authula/authula/plugins/totp/repository"
	totptests "github.com/Authula/authula/plugins/totp/tests"
	"github.com/Authula/authula/plugins/totp/types"
)

// newEmptyTestDB creates an in-memory SQLite database without any migrations,
// so repository queries fail the way they would before the schema is applied.
func newEmptyTestDB(t *testing.T) *bun.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func newHookPlugin(t *testing.T) (*TOTPPlugin, *bun.DB, *repository.TOTPRepository, *tests.MockTokenService, *tests.MockVerificationService) {
	t.Helper()

	plugin, _, tokenSvc, verifSvc := buildTestPlugin(t)
	db := newHookTestDB(t)
	repo := repository.NewTOTPRepository(db)
	plugin.totpRepo = repo

	return plugin, db, repo, tokenSvc, verifSvc
}

func seedUser(t *testing.T, db *bun.DB, userID string) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		`INSERT INTO users (id, name, email) VALUES (?, ?, ?)`,
		userID,
		"User "+userID,
		userID+"@example.com",
	)
	require.NoError(t, err)
}

func seedEnabledTOTP(t *testing.T, db *bun.DB, repo *repository.TOTPRepository, userID string) {
	t.Helper()

	ctx := context.Background()
	seedUser(t, db, userID)

	_, err := repo.Create(ctx, userID, "enc-secret", "[]")
	require.NoError(t, err)
	require.NoError(t, repo.SetEnabled(ctx, userID, true))
}

func TestSignInSuccessMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{name: "missing value", values: map[string]any{}, want: false},
		{name: "false value", values: map[string]any{models.ContextAuthSuccess.String(): false}, want: false},
		{name: "true value", values: map[string]any{models.ContextAuthSuccess.String(): true}, want: true},
		{name: "non-boolean value", values: map[string]any{models.ContextAuthSuccess.String(): "true"}, want: false},
	}

	plugin := New(types.TOTPPluginConfig{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reqCtx := &models.RequestContext{Values: tt.values}
			assert.Equal(t, tt.want, plugin.signInSuccessMatcher(reqCtx))
		})
	}
}

func TestInterceptSignInHookBranches(t *testing.T) {
	t.Parallel()

	type scenario struct {
		name              string
		path              string
		actor             *models.Actor
		prepare           func(t *testing.T, plugin *TOTPPlugin, db *bun.DB, repo *repository.TOTPRepository, tokenSvc *tests.MockTokenService, verifSvc *tests.MockVerificationService)
		wantStatus        int
		wantHandled       bool
		wantPendingCookie bool
	}

	cases := []scenario{
		{
			name:  "verify route is skipped",
			path:  "/auth/totp/verify",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
		},
		{
			name: "nil actor passes through",
			path: "/auth/sign-in",
		},
		{
			name:  "actor without ID passes through",
			path:  "/auth/sign-in",
			actor: &models.Actor{Type: models.ActorUser},
		},
		{
			name:  "disabled totp passes through",
			path:  "/auth/sign-in",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			prepare: func(t *testing.T, _ *TOTPPlugin, db *bun.DB, _ *repository.TOTPRepository, _ *tests.MockTokenService, _ *tests.MockVerificationService) {
				t.Helper()
				seedUser(t, db, "user-1")
			},
		},
		{
			name:  "repository error returns service unavailable",
			path:  "/auth/sign-in",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			prepare: func(t *testing.T, plugin *TOTPPlugin, _ *bun.DB, _ *repository.TOTPRepository, _ *tests.MockTokenService, _ *tests.MockVerificationService) {
				t.Helper()
				plugin.totpRepo = repository.NewTOTPRepository(newEmptyTestDB(t))
			},
			wantStatus:  http.StatusServiceUnavailable,
			wantHandled: true,
		},
		{
			name:  "pending token generation failure returns service unavailable",
			path:  "/auth/sign-in",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			prepare: func(t *testing.T, _ *TOTPPlugin, db *bun.DB, repo *repository.TOTPRepository, tokenSvc *tests.MockTokenService, _ *tests.MockVerificationService) {
				t.Helper()
				seedEnabledTOTP(t, db, repo, "user-1")
				tokenSvc.On("Generate").Return(nil, errors.New("token failure")).Once()
			},
			wantStatus:  http.StatusServiceUnavailable,
			wantHandled: true,
		},
		{
			name:  "pending verification failure returns service unavailable",
			path:  "/auth/sign-in",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			prepare: func(t *testing.T, _ *TOTPPlugin, db *bun.DB, repo *repository.TOTPRepository, tokenSvc *tests.MockTokenService, verifSvc *tests.MockVerificationService) {
				t.Helper()
				seedEnabledTOTP(t, db, repo, "user-1")
				tokenSvc.On("Generate").Return("pending-token", nil).Once()
				tokenSvc.On("Hash", "pending-token").Return("hashed-pending").Once()
				verifSvc.On(
					"Create",
					mock.Anything,
					"user-1",
					"hashed-pending",
					models.TypeTOTPPendingAuth,
					"user-1",
					mock.Anything,
				).Return(nil, errors.New("verification failure")).Once()
			},
			wantStatus:  http.StatusServiceUnavailable,
			wantHandled: true,
		},
		{
			name:  "pending auth response issues cookie and clears session values",
			path:  "/auth/sign-in",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			prepare: func(t *testing.T, _ *TOTPPlugin, db *bun.DB, repo *repository.TOTPRepository, tokenSvc *tests.MockTokenService, verifSvc *tests.MockVerificationService) {
				t.Helper()
				seedEnabledTOTP(t, db, repo, "user-1")
				tokenSvc.On("Generate").Return("pending-token", nil).Once()
				tokenSvc.On("Hash", "pending-token").Return("hashed-pending").Once()
				verifSvc.On(
					"Create",
					mock.Anything,
					"user-1",
					"hashed-pending",
					models.TypeTOTPPendingAuth,
					"user-1",
					mock.Anything,
				).Return(&models.Verification{ID: "verif-1"}, nil).Once()
			},
			wantStatus:        http.StatusOK,
			wantPendingCookie: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin, db, repo, tokenSvc, verifSvc := newHookPlugin(t)
			if tt.prepare != nil {
				tt.prepare(t, plugin, db, repo, tokenSvc, verifSvc)
			}

			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			w := httptest.NewRecorder()
			reqCtx := &models.RequestContext{
				Request:        req,
				ResponseWriter: w,
				Path:           tt.path,
				Method:         http.MethodPost,
				Actor:          tt.actor,
				Values: map[string]any{
					models.ContextAuthSuccess.String():  true,
					models.ContextSessionID.String():    "session-1",
					models.ContextSessionToken.String(): "session-token",
				},
			}

			err := plugin.interceptSignInHook(reqCtx)
			require.NoError(t, err)

			assert.Equal(t, tt.wantHandled, reqCtx.Handled)
			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.Equal(t, tt.wantStatus != 0, reqCtx.ResponseReady)

			if tt.wantPendingCookie {
				pendingCookieValue, ok := findPendingCookieValue(w)
				require.True(t, ok)
				assert.Equal(t, "pending-token", pendingCookieValue)

				cookie := totptests.CookieFromRecorder(w, constants.CookieTOTPPending)
				require.NotNil(t, cookie)
				assert.Equal(t, "/", cookie.Path)
				assert.Equal(t, int(plugin.pluginConfig.PendingTokenExpiry.Seconds()), cookie.MaxAge)
				assert.True(t, cookie.HttpOnly)
				assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)

				assert.NotContains(t, reqCtx.Values, models.ContextAuthSuccess.String())
				assert.NotContains(t, reqCtx.Values, models.ContextSessionID.String())
				assert.NotContains(t, reqCtx.Values, models.ContextSessionToken.String())
				assert.JSONEq(t, `{"totp_redirect":true}`, string(reqCtx.ResponseBody))
			}

			if tt.wantStatus == http.StatusServiceUnavailable {
				assert.Contains(t, string(reqCtx.ResponseBody), "totp verification unavailable")
			}

			tokenSvc.AssertExpectations(t)
			verifSvc.AssertExpectations(t)
		})
	}
}

func TestHasTrustedDevice(t *testing.T) {
	t.Parallel()

	type deviceSeed struct {
		userID    string
		token     string
		expiresAt time.Time
	}

	tests := []struct {
		name          string
		userID        string
		cookie        *http.Cookie
		hashedToken   string
		device        *deviceSeed
		want          bool
		wantRefreshed bool
	}{
		{
			name:   "no cookie",
			userID: "user-1",
		},
		{
			name:   "empty cookie value",
			userID: "user-1",
			cookie: &http.Cookie{Name: constants.CookieTOTPTrusted, Value: ""},
		},
		{
			name:        "unknown token",
			userID:      "user-1",
			cookie:      &http.Cookie{Name: constants.CookieTOTPTrusted, Value: "unknown-token"},
			hashedToken: "hash-unknown",
		},
		{
			name:        "device belongs to another user",
			userID:      "user-1",
			cookie:      &http.Cookie{Name: constants.CookieTOTPTrusted, Value: "other-token"},
			hashedToken: "hash-other",
			device: &deviceSeed{
				userID:    "user-2",
				token:     "hash-other",
				expiresAt: time.Now().UTC().Add(24 * time.Hour),
			},
		},
		{
			name:        "expired device",
			userID:      "user-1",
			cookie:      &http.Cookie{Name: constants.CookieTOTPTrusted, Value: "expired-token"},
			hashedToken: "hash-expired",
			device: &deviceSeed{
				userID:    "user-1",
				token:     "hash-expired",
				expiresAt: time.Now().UTC().Add(-time.Hour),
			},
		},
		{
			name:        "valid device is refreshed",
			userID:      "user-1",
			cookie:      &http.Cookie{Name: constants.CookieTOTPTrusted, Value: "valid-token"},
			hashedToken: "hash-valid",
			device: &deviceSeed{
				userID:    "user-1",
				token:     "hash-valid",
				expiresAt: time.Now().UTC().Add(24 * time.Hour),
			},
			want:          true,
			wantRefreshed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin, db, repo, tokenSvc, _ := newHookPlugin(t)
			ctx := context.Background()
			seedUser(t, db, "user-1")
			seedUser(t, db, "user-2")

			if tt.device != nil {
				_, err := repo.CreateTrustedDevice(ctx, tt.device.userID, tt.device.token, "ua", tt.device.expiresAt)
				require.NoError(t, err)
			}
			if tt.hashedToken != "" {
				tokenSvc.On("Hash", tt.cookie.Value).Return(tt.hashedToken).Once()
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			reqCtx := &models.RequestContext{Request: req}

			assert.Equal(t, tt.want, plugin.hasTrustedDevice(reqCtx, tt.userID))

			if tt.wantRefreshed {
				device, err := repo.GetTrustedDeviceByToken(ctx, tt.hashedToken)
				require.NoError(t, err)
				require.NotNil(t, device)
				assert.True(t, device.ExpiresAt.After(time.Now().UTC().Add(29*24*time.Hour)))
			}

			tokenSvc.AssertExpectations(t)
		})
	}
}
