package accesscontrol

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/usecases"
)

func newHookTestPluginWithUserAccessRepoMock(t *testing.T) (*AccessControlPlugin, interface {
	On(string, ...any) *mock.Call
	AssertExpectations(mock.TestingT) bool
}) {
	t.Helper()

	userAccessUseCase, userAccessRepo := tests.NewUserRolesUseCaseFixture()
	useCases := usecases.NewAccessControlUseCases(usecases.RolePermissionUseCase{}, userAccessUseCase)
	plugin := &AccessControlPlugin{Api: &API{useCases: useCases}}

	return plugin, userAccessRepo
}

func TestRequireAccessControlHook(t *testing.T) {
	t.Parallel()

	t.Run("unauthorized without user", func(t *testing.T) {
		t.Parallel()

		plugin := &AccessControlPlugin{}
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{Request: req}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.True(t, reqCtx.Handled, "expected request to be handled")
		require.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
	})

	t.Run("unauthorized with empty user id", func(t *testing.T) {
		t.Parallel()

		plugin := &AccessControlPlugin{}
		userID := ""
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{
			Request: req,
			UserID:  &userID,
		}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.True(t, reqCtx.Handled, "expected request to be handled")
		require.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
	})

	t.Run("opt in skips when no permissions metadata", func(t *testing.T) {
		t.Parallel()

		plugin := &AccessControlPlugin{}
		userID := "user-1"
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{
			Request: req,
			Route: &models.Route{
				Metadata: map[string]any{},
			},
			UserID: &userID,
		}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.False(t, reqCtx.Handled, "expected request not to be handled when no permissions metadata is set")
	})

	t.Run("internal error when HasPermissions fails", func(t *testing.T) {
		t.Parallel()

		plugin, userAccessRepo := newHookTestPluginWithUserAccessRepoMock(t)
		userID := "user-1"
		userAccessRepo.On("GetUserEffectivePermissions", mock.Anything, userID).Return(nil, errors.New("database error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{
			Request: req,
			Route: &models.Route{
				Metadata: map[string]any{
					"permissions": []string{"admin"},
				},
			},
			UserID: &userID,
		}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.True(t, reqCtx.Handled)
		require.Equal(t, http.StatusInternalServerError, reqCtx.ResponseStatus)
		userAccessRepo.AssertExpectations(t)
	})

	t.Run("forbidden when user lacks permissions", func(t *testing.T) {
		t.Parallel()

		plugin, userAccessRepo := newHookTestPluginWithUserAccessRepoMock(t)
		userID := "user-1"
		userAccessRepo.On("GetUserEffectivePermissions", mock.Anything, userID).Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "profiles.read"}}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{
			Request: req,
			Route: &models.Route{
				Metadata: map[string]any{
					"permissions": []string{"users.read"},
				},
			},
			UserID: &userID,
		}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.True(t, reqCtx.Handled, "expected request to be handled")
		require.Equal(t, http.StatusForbidden, reqCtx.ResponseStatus)
		userAccessRepo.AssertExpectations(t)
	})

	t.Run("allowed when user has permissions", func(t *testing.T) {
		t.Parallel()

		plugin, userAccessRepo := newHookTestPluginWithUserAccessRepoMock(t)
		userID := "user-1"
		userAccessRepo.On("GetUserEffectivePermissions", mock.Anything, userID).Return([]types.UserPermissionInfo{{PermissionKey: "users.read"}}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{
			Request: req,
			Route: &models.Route{
				Metadata: map[string]any{
					"permissions": []string{"users.read"},
				},
			},
			UserID: &userID,
		}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.False(t, reqCtx.Handled, "expected request not to be handled when user has permissions")
		userAccessRepo.AssertExpectations(t)
	})

	t.Run("multiple permissions allow when user has any one required permission", func(t *testing.T) {
		t.Parallel()

		plugin, userAccessRepo := newHookTestPluginWithUserAccessRepoMock(t)
		userID := "user-1"
		userAccessRepo.On("GetUserEffectivePermissions", mock.Anything, userID).Return([]types.UserPermissionInfo{{PermissionKey: "users.read"}}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{
			Request: req,
			Route: &models.Route{
				Metadata: map[string]any{
					"permissions": []string{"users.read", "users.write"},
				},
			},
			UserID: &userID,
		}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.False(t, reqCtx.Handled, "expected request not to be handled when user has at least one required permission")
		userAccessRepo.AssertExpectations(t)
	})

	t.Run("permissions metadata with whitespace trimming", func(t *testing.T) {
		t.Parallel()

		plugin, userAccessRepo := newHookTestPluginWithUserAccessRepoMock(t)
		userID := "user-1"
		userAccessRepo.On("GetUserEffectivePermissions", mock.Anything, userID).Return([]types.UserPermissionInfo{{PermissionKey: "users.read"}}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		reqCtx := &models.RequestContext{
			Request: req,
			Route: &models.Route{
				Metadata: map[string]any{
					"permissions": []string{"  users.read  ", "   ", ""},
				},
			},
			UserID: &userID,
		}

		err := plugin.requireAccessControl(reqCtx)
		require.NoError(t, err)
		require.False(t, reqCtx.Handled, "expected request not to be handled when user has permissions (after trimming)")
		userAccessRepo.AssertExpectations(t)
	})
}
