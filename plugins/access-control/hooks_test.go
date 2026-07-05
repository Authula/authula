package accesscontrol

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	authmodels "github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/services"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type noopAuthorizer struct{}

func (a *noopAuthorizer) AuthorizeScope(_ context.Context, _ *authmodels.Actor, _ string) error {
	return nil
}

func (a *noopAuthorizer) AuthorizeOrganizationAccess(_ context.Context, _ *authmodels.Actor, _ string, _ string) error {
	return nil
}

func newAccessControlHookTestPlugin(logger authmodels.Logger, rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) *AccessControlPlugin {
	authorizer := &noopAuthorizer{}
	rolesService := services.NewRolesService(rolesRepo, nil, userRolesRepo, authorizer)
	userRolesService := services.NewUserRolesService(userRolesRepo, rolesRepo, authorizer)
	accessControlService := services.NewAccessControlService(rolesService, userRolesService, nil)
	rolePermissionsService := services.NewRolePermissionsService(nil, nil, nil, authorizer)
	useCases := usecases.NewAccessControlUseCases(
		usecases.NewRolesUseCase(rolesService),
		usecases.NewPermissionsUseCase(nil),
		usecases.NewRolePermissionsUseCase(rolePermissionsService),
		usecases.NewUserRolesUseCase(userRolesService),
		usecases.NewUserPermissionsUseCase(nil),
	)

	return &AccessControlPlugin{
		Api:                  NewAPI(useCases),
		logger:               logger,
		accessControlService: accessControlService,
	}
}

func TestHasScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []string
		target string
		want   bool
	}{
		{name: "exact_match", scopes: []string{"admin:users:list"}, target: "admin:users:list", want: true},
		{name: "universal_wildcard", scopes: []string{"*"}, target: "admin:users:list", want: true},
		{name: "prefix_wildcard_match", scopes: []string{"admin:*"}, target: "admin:users:list", want: true},
		{name: "prefix_wildcard_nested", scopes: []string{"admin:users:*"}, target: "admin:users:create", want: true},
		{name: "prefix_wildcard_no_match", scopes: []string{"admin:*"}, target: "users:list", want: false},
		{name: "no_match", scopes: []string{"admin:users:create"}, target: "admin:users:list", want: false},
		{name: "empty_scopes", scopes: []string{}, target: "admin:users:list", want: false},
		{name: "multiple_scopes_wildcard_match", scopes: []string{"org:read", "admin:*"}, target: "admin:users:list", want: true},
		{name: "prefix_wildcard_middle", scopes: []string{"admin:auth:*"}, target: "admin:auth:login", want: true},
		{name: "prefix_wildcard_wrong_domain", scopes: []string{"admin:auth:*"}, target: "admin:users:list", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := hasScope(tc.scopes, tc.target)
			if got != tc.want {
				t.Errorf("hasScope(%v, %q) = %v, want %v", tc.scopes, tc.target, got, tc.want)
			}
		})
	}
}

func TestHasAllScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		assignedScopes []string
		requiredScopes []string
		want           bool
	}{
		{name: "all_exact_match", assignedScopes: []string{"read", "write"}, requiredScopes: []string{"read", "write"}, want: true},
		{name: "wildcard_covers_multiple", assignedScopes: []string{"admin:*"}, requiredScopes: []string{"admin:users:list", "admin:users:create"}, want: true},
		{name: "mixed_wildcard_and_exact", assignedScopes: []string{"read", "admin:*"}, requiredScopes: []string{"read", "admin:users:list"}, want: true},
		{name: "missing_permission", assignedScopes: []string{"admin:*"}, requiredScopes: []string{"admin:users:list", "users:list"}, want: false},
		{name: "empty_required", assignedScopes: []string{"admin:*"}, requiredScopes: []string{}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := hasAllScopes(tc.assignedScopes, tc.requiredScopes)
			if got != tc.want {
				t.Errorf("hasAllScopes(%v, %v) = %v, want %v", tc.assignedScopes, tc.requiredScopes, got, tc.want)
			}
		})
	}
}

func TestAccessControlPluginHooksIncludesGlobalAssignRoleHook(t *testing.T) {
	t.Parallel()

	hooks := (&AccessControlPlugin{}).Hooks()
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(hooks))
	}

	var foundGlobal bool
	for _, hook := range hooks {
		if hook.Stage == authmodels.HookAfter && hook.PluginID == "" && hook.Handler != nil {
			foundGlobal = true
		}
	}

	if !foundGlobal {
		t.Fatal("expected a global HookAfter assignment hook")
	}
}

func TestAccessControlPluginAssignRoleFromContextHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		contextValue any
		setup        func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
	}{
		{
			name:         "missing context is a no-op",
			contextValue: nil,
		},
		{
			name:         "already assigned role is skipped",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor"},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "Editor").Return(&types.Role{ID: "role-1", Name: "Editor", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-1", RoleName: "Editor", RoleWeight: 10}}, nil).Once()
			},
		},
		{
			name:         "assigns role when missing",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor", AssignerUserID: func() *string { v := "assigner-1"; return &v }()},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "Editor").Return(&types.Role{ID: "role-1", Name: "Editor", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", mock.MatchedBy(func(userID *string) bool {
					return userID != nil && *userID == "assigner-1"
				}), (*time.Time)(nil)).Return(nil).Once()
			},
		},
		{
			name:         "role lookup failure is logged and ignored",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor", AssignerUserID: func() *string { v := "assigner-1"; return &v }()},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "Editor").Return((*types.Role)(nil), errors.New("lookup failed")).Once()
			},
		},
		{
			name:         "assignment failure is logged and ignored",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor"},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "Editor").Return(&types.Role{ID: "role-1", Name: "Editor", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(errors.New("assign failed")).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo, userRolesRepo)
			}

			plugin := newAccessControlHookTestPlugin(&internaltests.MockLogger{}, rolesRepo, userRolesRepo)

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			reqCtx := &authmodels.RequestContext{
				Request: req,
				Values:  map[string]any{},
			}
			if tc.contextValue != nil {
				reqCtx.Values[authmodels.ContextAccessControlAssignRole.String()] = tc.contextValue
			}

			err := plugin.assignRoleFromContextHook(reqCtx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}
