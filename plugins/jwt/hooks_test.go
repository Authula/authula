package jwt

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

var errHookTest = errors.New("hook test error")

func newTestPlugin(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) *JWTPlugin {
	return newTestPluginWithBlacklist(tokenSvc, refreshSvc, nil)
}

func newTestPluginWithBlacklist(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService, blacklistSvc *jwttests.MockBlacklistService) *JWTPlugin {
	return &JWTPlugin{
		pluginConfig: types.JWTPluginConfig{
			ExpiresIn:        15 * time.Minute,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
		jwtService:       tokenSvc,
		refreshService:   refreshSvc,
		blacklistService: blacklistSvc,
		Logger:           &internaltests.MockLogger{},
	}
}

func matchesTimeNear(target time.Time, tolerance time.Duration) func(time.Time) bool {
	return func(actual time.Time) bool {
		delta := actual.Sub(target)
		return delta >= -tolerance && delta <= tolerance
	}
}

func TestIssueTokensHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		actor   *models.Actor
		setup   func(*testing.T, *models.RequestContext)
		mock    func(*jwttests.MockTokenService, *jwttests.MockRefreshTokenService)
		wantErr string
		check   func(*testing.T, *models.RequestContext)
	}{
		{
			name:  "nil_actor",
			actor: nil,
			mock:  func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.Empty(t, reqCtx.Values)
			},
		},
		{
			name:  "skip_mint_flag",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(t *testing.T, reqCtx *models.RequestContext) {
				reqCtx.Values[models.ContextAuthIdempotentSkipTokensMint.String()] = true
			},
			mock: func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.Empty(t, reqCtx.Values[types.JWTTokenTypeAccess.String()])
			},
		},
		{
			name:  "user_success",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(t *testing.T, reqCtx *models.RequestContext) {
				reqCtx.Values[models.ContextSessionID.String()] = "sess-1"
			},
			mock: func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {
				pair := &types.TokenPair{
					AccessToken:  "access-token-1",
					RefreshToken: "refresh-token-1",
				}
				tokenSvc.On("GenerateUserToken", mock.Anything, "user-1", "sess-1", mock.Anything).Return(pair, nil)
				refreshSvc.On("StoreInitialRefreshToken", mock.Anything, "refresh-token-1", "sess-1", mock.Anything).Return(nil)
			},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.Equal(t, "access-token-1", reqCtx.Values[types.JWTTokenTypeAccess.String()])
				require.Equal(t, "refresh-token-1", reqCtx.Values[types.JWTTokenTypeRefresh.String()])
			},
		},
		{
			name:  "user_no_session_id",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			mock:  func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.Empty(t, reqCtx.Values)
			},
		},
		{
			name:  "user_token_error",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(t *testing.T, reqCtx *models.RequestContext) {
				reqCtx.Values[models.ContextSessionID.String()] = "sess-1"
			},
			mock: func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {
				tokenSvc.On("GenerateUserToken", mock.Anything, "user-1", "sess-1", mock.Anything).Return(nil, errHookTest)
			},
			wantErr: "failed to generate authentication tokens",
		},
		{
			name: "machine_success",
			actor: &models.Actor{
				ID:     "client-1",
				Type:   models.ActorMachine,
				Claims: map[string]any{"organization_id": "org-1"},
				Scopes: []string{"read", "write"},
			},
			mock: func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {
				pair := &types.TokenPair{AccessToken: "machine-access-token"}
				tokenSvc.On("GenerateMachineToken", mock.Anything, "client-1", "org-1", []string{"read", "write"}).Return(pair, nil)
			},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.Equal(t, "machine-access-token", reqCtx.Values[types.JWTTokenTypeAccess.String()])
				_, hasRefresh := reqCtx.Values[types.JWTTokenTypeRefresh.String()]
				require.False(t, hasRefresh)
			},
		},
		{
			name: "machine_no_optional_fields",
			actor: &models.Actor{
				ID:     "client-2",
				Type:   models.ActorMachine,
				Claims: map[string]any{"organization_id": "org-2"},
			},
			mock: func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {
				pair := &types.TokenPair{AccessToken: "machine-access-token-2"}
				tokenSvc.On("GenerateMachineToken", mock.Anything, "client-2", "org-2", []string(nil)).Return(pair, nil)
			},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.Equal(t, "machine-access-token-2", reqCtx.Values[types.JWTTokenTypeAccess.String()])
			},
		},
		{
			name: "machine_token_error",
			actor: &models.Actor{
				ID:     "client-3",
				Type:   models.ActorMachine,
				Claims: map[string]any{"organization_id": "org-3"},
			},
			mock: func(tokenSvc *jwttests.MockTokenService, refreshSvc *jwttests.MockRefreshTokenService) {
				tokenSvc.On("GenerateMachineToken", mock.Anything, "client-3", "org-3", []string(nil)).Return(nil, errHookTest)
			},
			wantErr: "failed to generate machine authentication tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenSvc := new(jwttests.MockTokenService)
			refreshSvc := new(jwttests.MockRefreshTokenService)

			_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, tt.actor)

			if tt.setup != nil {
				tt.setup(t, reqCtx)
			}

			if tt.mock != nil {
				tt.mock(tokenSvc, refreshSvc)
			}

			plugin := newTestPlugin(tokenSvc, refreshSvc)

			err := plugin.issueTokensHook(reqCtx)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.check != nil {
				tt.check(t, reqCtx)
			}

			tokenSvc.AssertExpectations(t)
			refreshSvc.AssertExpectations(t)
		})
	}
}

func TestRevokeSessionOnSignOutHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		values       map[string]any
		blacklistErr error
		revokeErr    error
		wantCalls    bool
		wantLog      string
	}{
		{
			name: "sign-out_blacklists_and_revokes_current_session",
			values: map[string]any{
				models.ContextAuthSignOut.String(): true,
				models.ContextSessionID.String():   "sess-1",
			},
			wantCalls: true,
		},
		{
			name: "sign-out_without_session_id_is_noop",
			values: map[string]any{
				models.ContextAuthSignOut.String(): true,
			},
		},
		{
			name: "sign-out_continues_when_blacklist_fails",
			values: map[string]any{
				models.ContextAuthSignOut.String(): true,
				models.ContextSessionID.String():   "sess-1",
			},
			blacklistErr: errHookTest,
			wantCalls:    true,
			wantLog:      "failed to blacklist session tokens on sign-out",
		},
		{
			name: "sign-out_continues_when_revoke_fails",
			values: map[string]any{
				models.ContextAuthSignOut.String(): true,
				models.ContextSessionID.String():   "sess-1",
			},
			revokeErr: errHookTest,
			wantCalls: true,
			wantLog:   "failed to revoke session refresh tokens on sign-out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenSvc := new(jwttests.MockTokenService)
			refreshSvc := new(jwttests.MockRefreshTokenService)
			blacklistSvc := new(jwttests.MockBlacklistService)
			logger := &internaltests.LevelRecordingLogger{}

			_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodPost, "/sign-out", nil, nil)
			for key, value := range tt.values {
				reqCtx.Values[key] = value
			}

			plugin := newTestPluginWithBlacklist(tokenSvc, refreshSvc, blacklistSvc)
			plugin.Logger = logger

			require.True(t, plugin.signedOutMatcher(reqCtx))

			expiresAt := time.Now().Add(plugin.pluginConfig.RefreshExpiresIn)
			if tt.wantCalls {
				blacklistSvc.On("BlacklistAllSessionTokens", mock.Anything, "sess-1", mock.MatchedBy(matchesTimeNear(expiresAt, 5*time.Second))).Return(tt.blacklistErr)
				refreshSvc.On("RevokeSessionTokens", mock.Anything, "sess-1").Return(tt.revokeErr)
			}

			err := plugin.revokeSessionOnSignOutHook(reqCtx)
			require.NoError(t, err)

			if tt.wantCalls {
				blacklistSvc.AssertCalled(t, "BlacklistAllSessionTokens", mock.Anything, "sess-1", mock.Anything)
				refreshSvc.AssertCalled(t, "RevokeSessionTokens", mock.Anything, "sess-1")
			} else {
				blacklistSvc.AssertNotCalled(t, "BlacklistAllSessionTokens", mock.Anything, mock.Anything, mock.Anything)
				refreshSvc.AssertNotCalled(t, "RevokeSessionTokens", mock.Anything, mock.Anything)
			}

			if tt.wantLog != "" {
				require.Contains(t, logger.Messages(), tt.wantLog)
			}

			blacklistSvc.AssertExpectations(t)
			refreshSvc.AssertExpectations(t)
		})
	}
}

func TestSignedOutMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{
			name:   "true sign out matches",
			values: map[string]any{models.ContextAuthSignOut.String(): true},
			want:   true,
		},
		{
			name:   "false sign out does not match",
			values: map[string]any{models.ContextAuthSignOut.String(): false},
		},
		{
			name:   "non boolean sign out does not match",
			values: map[string]any{models.ContextAuthSignOut.String(): "true"},
		},
		{
			name:   "missing sign out does not match",
			values: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodPost, "/sign-out", nil, nil)
			for key, value := range tt.values {
				reqCtx.Values[key] = value
			}

			plugin := newTestPlugin(new(jwttests.MockTokenService), new(jwttests.MockRefreshTokenService))
			require.Equal(t, tt.want, plugin.signedOutMatcher(reqCtx))
		})
	}
}

func TestRespondHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		actor   *models.Actor
		setup   func(*testing.T, *models.RequestContext)
		wantErr string
		check   func(*testing.T, *models.RequestContext)
	}{
		{
			name:  "nil_actor",
			actor: nil,
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.False(t, reqCtx.Handled)
			},
		},
		{
			name:  "no_access_token",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.False(t, reqCtx.Handled)
			},
		},
		{
			name:  "redirect_is_not_overridden",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(t *testing.T, reqCtx *models.RequestContext) {
				reqCtx.RedirectURL = "/welcome"
				reqCtx.Values[types.JWTTokenTypeAccess.String()] = "access-1"
			},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.False(t, reqCtx.Handled)
				require.Empty(t, reqCtx.ResponseBody)
				require.Equal(t, "/welcome", reqCtx.RedirectURL)
			},
		},
		{
			name:  "already_handled_is_not_overridden",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(t *testing.T, reqCtx *models.RequestContext) {
				reqCtx.Handled = true
				reqCtx.Values[types.JWTTokenTypeAccess.String()] = "access-1"
			},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.True(t, reqCtx.Handled)
				require.Empty(t, reqCtx.ResponseBody)
			},
		},
		{
			name:  "user_success",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(t *testing.T, reqCtx *models.RequestContext) {
				reqCtx.Values[types.JWTTokenTypeAccess.String()] = "access-1"
				reqCtx.Values[types.JWTTokenTypeRefresh.String()] = "refresh-1"
			},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.True(t, reqCtx.Handled)
				require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
				require.JSONEq(t, `{"access_token":"access-1","token_type":"Bearer","refresh_token":"refresh-1","expires_in":900}`, string(reqCtx.ResponseBody))
				require.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
				require.Equal(t, "no-cache", reqCtx.ResponseHeaders.Get("Pragma"))
			},
		},
		{
			name: "machine_success",
			actor: &models.Actor{
				ID:     "client-1",
				Type:   models.ActorMachine,
				Claims: map[string]any{"organization_id": "org-1"},
			},
			setup: func(t *testing.T, reqCtx *models.RequestContext) {
				reqCtx.Values[types.JWTTokenTypeAccess.String()] = "machine-access-1"
			},
			check: func(t *testing.T, reqCtx *models.RequestContext) {
				require.True(t, reqCtx.Handled)
				require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
				require.JSONEq(t, `{"access_token":"machine-access-1","token_type":"Bearer","expires_in":900}`, string(reqCtx.ResponseBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, tt.actor)

			if tt.setup != nil {
				tt.setup(t, reqCtx)
			}

			plugin := newTestPlugin(new(jwttests.MockTokenService), new(jwttests.MockRefreshTokenService))

			err := plugin.respondHook(reqCtx)

			if tt.wantErr != "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.check != nil {
				tt.check(t, reqCtx)
			}
		})
	}
}
