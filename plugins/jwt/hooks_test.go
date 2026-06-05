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
	return &JWTPlugin{
		pluginConfig: types.JWTPluginConfig{
			ExpiresIn:        15 * time.Minute,
			RefreshExpiresIn: 7 * 24 * time.Hour,
		},
		jwtService:     tokenSvc,
		refreshService: refreshSvc,
		Logger:         &mockLogger{},
	}
}

func TestIssueTokensHook(t *testing.T) {
	t.Parallel()

	t.Run("nil_actor", func(t *testing.T) {
		t.Parallel()

		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, nil)
		plugin := newTestPlugin(&jwttests.MockTokenService{}, &jwttests.MockRefreshTokenService{})

		err := plugin.issueTokensHook(reqCtx)
		require.NoError(t, err)
		require.Empty(t, reqCtx.Values)
	})

	t.Run("skip_mint_flag", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)
		reqCtx.Values[models.ContextAuthIdempotentSkipTokensMint.String()] = true

		plugin := newTestPlugin(&jwttests.MockTokenService{}, &jwttests.MockRefreshTokenService{})

		err := plugin.issueTokensHook(reqCtx)
		require.NoError(t, err)
		require.Empty(t, reqCtx.Values[types.JWTTokenTypeAccess.String()])
	})

	t.Run("user_success", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)
		reqCtx.Values[models.ContextSessionID.String()] = "sess-1"

		tokenSvc := &jwttests.MockTokenService{}
		refreshSvc := &jwttests.MockRefreshTokenService{}
		pair := &types.TokenPair{
			AccessToken:  "access-token-1",
			RefreshToken: "refresh-token-1",
		}
		tokenSvc.On("GenerateUserToken", mock.Anything, "user-1", "sess-1").Return(pair, nil)
		refreshSvc.On("StoreInitialRefreshToken", mock.Anything, "refresh-token-1", "sess-1", mock.Anything).Return(nil)

		plugin := newTestPlugin(tokenSvc, refreshSvc)

		err := plugin.issueTokensHook(reqCtx)
		require.NoError(t, err)
		require.Equal(t, "access-token-1", reqCtx.Values[types.JWTTokenTypeAccess.String()])
		require.Equal(t, "refresh-token-1", reqCtx.Values[types.JWTTokenTypeRefresh.String()])

		tokenSvc.AssertExpectations(t)
		refreshSvc.AssertExpectations(t)
	})

	t.Run("user_no_session_id", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)

		plugin := newTestPlugin(&jwttests.MockTokenService{}, &jwttests.MockRefreshTokenService{})

		err := plugin.issueTokensHook(reqCtx)
		require.NoError(t, err)
		require.Empty(t, reqCtx.Values)
	})

	t.Run("user_token_error", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)
		reqCtx.Values[models.ContextSessionID.String()] = "sess-1"

		tokenSvc := &jwttests.MockTokenService{}
		tokenSvc.On("GenerateUserToken", mock.Anything, "user-1", "sess-1").Return(nil, errHookTest)

		plugin := newTestPlugin(tokenSvc, &jwttests.MockRefreshTokenService{})

		err := plugin.issueTokensHook(reqCtx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to generate authentication tokens")

		tokenSvc.AssertExpectations(t)
	})

	t.Run("machine_success", func(t *testing.T) {
		t.Parallel()

		orgID := "org-1"
		actor := &models.Actor{
			ID:             "client-1",
			Type:           models.ActorMachine,
			OrganizationID: &orgID,
			Scopes:         []string{"read", "write"},
		}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)

		tokenSvc := &jwttests.MockTokenService{}
		pair := &types.TokenPair{AccessToken: "machine-access-token"}
		tokenSvc.On("GenerateMachineToken", mock.Anything, "client-1", "org-1", []string{"read", "write"}).Return(pair, nil)

		plugin := newTestPlugin(tokenSvc, &jwttests.MockRefreshTokenService{})

		err := plugin.issueTokensHook(reqCtx)
		require.NoError(t, err)
		require.Equal(t, "machine-access-token", reqCtx.Values[types.JWTTokenTypeAccess.String()])
		_, hasRefresh := reqCtx.Values[types.JWTTokenTypeRefresh.String()]
		require.False(t, hasRefresh)

		tokenSvc.AssertExpectations(t)
	})

	t.Run("machine_no_optional_fields", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{
			ID:   "client-2",
			Type: models.ActorMachine,
		}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)

		tokenSvc := &jwttests.MockTokenService{}
		pair := &types.TokenPair{AccessToken: "machine-access-token-2"}
		tokenSvc.On("GenerateMachineToken", mock.Anything, "client-2", "", []string(nil)).Return(pair, nil)

		plugin := newTestPlugin(tokenSvc, &jwttests.MockRefreshTokenService{})

		err := plugin.issueTokensHook(reqCtx)
		require.NoError(t, err)
		require.Equal(t, "machine-access-token-2", reqCtx.Values[types.JWTTokenTypeAccess.String()])

		tokenSvc.AssertExpectations(t)
	})

	t.Run("machine_token_error", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{
			ID:   "client-3",
			Type: models.ActorMachine,
		}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)

		tokenSvc := &jwttests.MockTokenService{}
		tokenSvc.On("GenerateMachineToken", mock.Anything, "client-3", "", []string(nil)).Return(nil, errHookTest)

		plugin := newTestPlugin(tokenSvc, &jwttests.MockRefreshTokenService{})

		err := plugin.issueTokensHook(reqCtx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to generate machine authentication tokens")

		tokenSvc.AssertExpectations(t)
	})
}

func TestRespondHook(t *testing.T) {
	t.Parallel()

	t.Run("nil_actor", func(t *testing.T) {
		t.Parallel()

		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, nil)
		plugin := newTestPlugin(&jwttests.MockTokenService{}, &jwttests.MockRefreshTokenService{})

		err := plugin.respondHook(reqCtx)
		require.NoError(t, err)
		require.False(t, reqCtx.Handled)
	})

	t.Run("no_access_token", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)

		plugin := newTestPlugin(&jwttests.MockTokenService{}, &jwttests.MockRefreshTokenService{})

		err := plugin.respondHook(reqCtx)
		require.NoError(t, err)
		require.False(t, reqCtx.Handled)
	})

	t.Run("user_success", func(t *testing.T) {
		t.Parallel()

		actor := &models.Actor{ID: "user-1", Type: models.ActorUser}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)
		reqCtx.Values[types.JWTTokenTypeAccess.String()] = "access-1"
		reqCtx.Values[types.JWTTokenTypeRefresh.String()] = "refresh-1"

		plugin := newTestPlugin(&jwttests.MockTokenService{}, &jwttests.MockRefreshTokenService{})

		err := plugin.respondHook(reqCtx)
		require.NoError(t, err)
		require.True(t, reqCtx.Handled)
		require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		require.JSONEq(t, `{"access_token":"access-1","token_type":"Bearer","refresh_token":"refresh-1"}`, string(reqCtx.ResponseBody))
	})

	t.Run("machine_success", func(t *testing.T) {
		t.Parallel()

		orgID := "org-1"
		actor := &models.Actor{
			ID:             "client-1",
			Type:           models.ActorMachine,
			OrganizationID: &orgID,
		}
		_, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, actor)
		reqCtx.Values[types.JWTTokenTypeAccess.String()] = "machine-access-1"

		plugin := newTestPlugin(&jwttests.MockTokenService{}, &jwttests.MockRefreshTokenService{})

		err := plugin.respondHook(reqCtx)
		require.NoError(t, err)
		require.True(t, reqCtx.Handled)
		require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
		require.JSONEq(t, `{"access_token":"machine-access-1","token_type":"Bearer"}`, string(reqCtx.ResponseBody))
	})
}

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any)  {}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}
