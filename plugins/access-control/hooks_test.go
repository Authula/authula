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

func (a *noopAuthorizer) AuthorizeOrganizationAccess(_ context.Context, _ *authmodels.Actor, _ string) error {
	return nil
}

func newAccessControlHookTestPlugin(logger authmodels.Logger, rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) *AccessControlPlugin {
	authorizer := &noopAuthorizer{}
	rolesService := services.NewRolesService(rolesRepo, nil, userRolesRepo)
	userRolesService := services.NewUserRolesService(userRolesRepo, rolesRepo)
	accessControlService := services.NewAccessControlService(rolesService, userRolesService, nil)
	rolePermissionsService := services.NewRolePermissionsService(nil, nil, nil)
	useCases := usecases.NewAccessControlUseCases(
		usecases.NewRolesUseCase(rolesService, authorizer),
		usecases.NewPermissionsUseCase(nil, authorizer),
		usecases.NewRolePermissionsUseCase(rolePermissionsService, authorizer),
		usecases.NewUserRolesUseCase(userRolesService, authorizer),
		usecases.NewUserPermissionsUseCase(nil, authorizer),
	)

	return &AccessControlPlugin{
		Api:                  NewAPI(rolesService, nil, rolePermissionsService, userRolesService, nil),
		useCases:             useCases,
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

func TestHasAllExactScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		assignedScopes []string
		requiredScopes []string
		want           bool
	}{
		{name: "all_exact_match", assignedScopes: []string{"read", "write"}, requiredScopes: []string{"read", "write"}, want: true},
		{name: "universal_wildcard_does_not_match", assignedScopes: []string{"*"}, requiredScopes: []string{"read"}, want: false},
		{name: "prefix_wildcard_does_not_match", assignedScopes: []string{"admin:*"}, requiredScopes: []string{"admin:users:list"}, want: false},
		{name: "missing_permission", assignedScopes: []string{"read"}, requiredScopes: []string{"read", "write"}, want: false},
		{name: "empty_required", assignedScopes: []string{}, requiredScopes: []string{}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := hasAllExactScopes(tc.assignedScopes, tc.requiredScopes)
			if got != tc.want {
				t.Errorf("hasAllExactScopes(%v, %v) = %v, want %v", tc.assignedScopes, tc.requiredScopes, got, tc.want)
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
			name:         "unsupported context value is a no-op",
			contextValue: "invalid",
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
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Editor", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-admin", RoleName: "admin", RoleWeight: 50}}, nil).Once()
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
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Editor", Weight: 10}, nil).Once()
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

func newAccessControlPermissionsHookTestPlugin(userPermissionsRepo *accesscontroltests.MockUserPermissionsRepository) *AccessControlPlugin {
	userPermissionsService := services.NewUserPermissionsService(userPermissionsRepo)

	return &AccessControlPlugin{
		Api:    NewAPI(nil, nil, nil, nil, userPermissionsService),
		logger: &internaltests.MockLogger{},
	}
}

func TestAccessControlPluginRequireAccessControl(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("permissions lookup failed")

	tests := []struct {
		name           string
		actor          *authmodels.Actor
		requiredScopes []string
		setup          func(*accesscontroltests.MockUserPermissionsRepository)
		wantErr        error
		wantHandled    bool
		wantStatus     int
		wantScopes     []string
	}{
		{
			name:  "nil actor is a no-op",
			actor: nil,
		},
		{
			name:  "actor without id is a no-op",
			actor: &authmodels.Actor{ID: ""},
		},
		{
			name:  "machine actor without required permissions is a no-op",
			actor: &authmodels.Actor{ID: "machine-1", Type: authmodels.ActorMachine},
		},
		{
			name:           "machine actor missing required permission is forbidden",
			actor:          &authmodels.Actor{ID: "machine-1", Type: authmodels.ActorMachine, Scopes: []string{"other:read"}},
			requiredScopes: []string{"users:read"},
			wantHandled:    true,
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "machine actor with exact permission is allowed",
			actor:          &authmodels.Actor{ID: "machine-1", Type: authmodels.ActorMachine, Scopes: []string{"users:read"}},
			requiredScopes: []string{"users:read"},
			wantScopes:     []string{"users:read"},
		},
		{
			name:           "machine actor wildcard scope is forbidden",
			actor:          &authmodels.Actor{ID: "machine-1", Type: authmodels.ActorMachine, Scopes: []string{"*"}},
			requiredScopes: []string{"users:read"},
			wantHandled:    true,
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "machine actor prefix wildcard scope is forbidden",
			actor:          &authmodels.Actor{ID: "machine-1", Type: authmodels.ActorMachine, Scopes: []string{"users:*"}},
			requiredScopes: []string{"users:read"},
			wantHandled:    true,
			wantStatus:     http.StatusForbidden,
		},
		{
			name:  "hydrates user scopes when empty",
			actor: &authmodels.Actor{ID: "user-1", Type: authmodels.ActorUser},
			setup: func(repo *accesscontroltests.MockUserPermissionsRepository) {
				repo.On("GetUserPermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionKey: "users:read"}}, nil).Once()
			},
			wantScopes: []string{"users:read"},
		},
		{
			name:           "grants access when required permission is present",
			actor:          &authmodels.Actor{ID: "user-1", Type: authmodels.ActorUser},
			requiredScopes: []string{"users:read"},
			setup: func(repo *accesscontroltests.MockUserPermissionsRepository) {
				repo.On("GetUserPermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionKey: "users:read"}}, nil).Once()
			},
			wantScopes: []string{"users:read"},
		},
		{
			name:           "denies access when required permission is missing",
			actor:          &authmodels.Actor{ID: "user-1", Type: authmodels.ActorUser},
			requiredScopes: []string{"users:write"},
			setup: func(repo *accesscontroltests.MockUserPermissionsRepository) {
				repo.On("GetUserPermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionKey: "users:read"}}, nil).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusForbidden,
			wantScopes:  []string{"users:read"},
		},
		{
			name:  "api key scopes remain when still active",
			actor: &authmodels.Actor{ID: "user-1", Type: authmodels.ActorUser, Scopes: []string{"users:read"}},
			setup: func(repo *accesscontroltests.MockUserPermissionsRepository) {
				repo.On("GetUserPermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionKey: "users:read"}, {PermissionKey: "users:write"}}, nil).Once()
			},
			wantScopes: []string{"users:read"},
		},
		{
			name:  "api key with stale scope is rejected",
			actor: &authmodels.Actor{ID: "user-1", Type: authmodels.ActorUser, Scopes: []string{"users:delete"}},
			setup: func(repo *accesscontroltests.MockUserPermissionsRepository) {
				repo.On("GetUserPermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionKey: "users:read"}}, nil).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:  "permission lookup failure returns internal error",
			actor: &authmodels.Actor{ID: "user-1", Type: authmodels.ActorUser},
			setup: func(repo *accesscontroltests.MockUserPermissionsRepository) {
				repo.On("GetUserPermissions", mock.Anything, "user-1").Return(([]types.UserPermissionInfo)(nil), repoErr).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userPermissionsRepo := &accesscontroltests.MockUserPermissionsRepository{}
			if tc.setup != nil {
				tc.setup(userPermissionsRepo)
			}

			plugin := newAccessControlPermissionsHookTestPlugin(userPermissionsRepo)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			reqCtx := &authmodels.RequestContext{
				Request: req,
				Actor:   tc.actor,
				Route:   &authmodels.Route{Metadata: map[string]any{}},
			}
			if tc.requiredScopes != nil {
				reqCtx.Route.Metadata["permissions"] = tc.requiredScopes
			}

			err := plugin.requireAccessControl(reqCtx)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if reqCtx.Handled != tc.wantHandled {
				t.Fatalf("expected handled=%v, got %v", tc.wantHandled, reqCtx.Handled)
			}
			if tc.wantStatus != 0 && reqCtx.ResponseStatus != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, reqCtx.ResponseStatus)
			}
			if tc.wantScopes != nil && tc.actor != nil {
				if len(tc.actor.Scopes) != len(tc.wantScopes) {
					t.Fatalf("expected scopes %v, got %v", tc.wantScopes, tc.actor.Scopes)
				}
				for i := range tc.wantScopes {
					if tc.actor.Scopes[i] != tc.wantScopes[i] {
						t.Fatalf("expected scopes %v, got %v", tc.wantScopes, tc.actor.Scopes)
					}
				}
			}

			userPermissionsRepo.AssertExpectations(t)
		})
	}
}

func TestAccessControlAssignRoleContext(t *testing.T) {
	t.Parallel()

	value := authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "editor"}

	tests := []struct {
		name   string
		input  any
		want   authmodels.AccessControlAssignRoleContext
		wantOK bool
	}{
		{
			name:   "struct value",
			input:  value,
			want:   value,
			wantOK: true,
		},
		{
			name:   "pointer value",
			input:  &value,
			want:   value,
			wantOK: true,
		},
		{
			name:   "nil pointer",
			input:  (*authmodels.AccessControlAssignRoleContext)(nil),
			wantOK: false,
		},
		{
			name:   "unsupported type",
			input:  "invalid",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := accessControlAssignRoleContext(tc.input)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("expected (%#v, %v), got (%#v, %v)", tc.want, tc.wantOK, got, ok)
			}
		})
	}
}

func TestExtractPermissionKeys(t *testing.T) {
	t.Parallel()

	permissions := []types.UserPermissionInfo{
		{PermissionKey: "users:read"},
		{PermissionKey: "users:write"},
	}

	keys := extractPermissionKeys(permissions)
	if len(keys) != 2 || keys[0] != "users:read" || keys[1] != "users:write" {
		t.Fatalf("unexpected keys %v", keys)
	}

	if keys := extractPermissionKeys(nil); len(keys) != 0 {
		t.Fatalf("expected empty keys, got %v", keys)
	}
}
