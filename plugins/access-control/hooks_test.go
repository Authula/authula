package accesscontrol

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	authmodels "github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/services"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type hookTestLogger struct {
	errors []string
}

func (l *hookTestLogger) Debug(msg string, args ...any) {}
func (l *hookTestLogger) Info(msg string, args ...any)  {}
func (l *hookTestLogger) Warn(msg string, args ...any)  {}
func (l *hookTestLogger) Error(msg string, args ...any) {
	l.errors = append(l.errors, msg+fmt.Sprint(args...))
}

func newAccessControlHookTestPlugin(logger authmodels.Logger, rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) *AccessControlPlugin {
	rolePermissionsService := services.NewRolePermissionsService(nil, nil, nil)
	useCases := usecases.NewAccessControlUseCases(
		usecases.NewRolesUseCase(services.NewRolesService(rolesRepo, nil, userRolesRepo)),
		usecases.NewPermissionsUseCase(nil),
		usecases.NewRolePermissionsUseCase(rolePermissionsService),
		usecases.NewUserRolesUseCase(services.NewUserRolesService(userRolesRepo, rolesRepo)),
		usecases.NewUserPermissionsUseCase(nil),
	)

	return &AccessControlPlugin{
		Api:    NewAPI(useCases),
		logger: logger,
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
		name           string
		contextValue   any
		setup          func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		wantErrors     int
		wantErrMessage string
	}{
		{
			name:         "missing context is a no-op",
			contextValue: nil,
		},
		{
			name:         "already assigned role is skipped",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor"},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-1", RoleName: "Editor"}}, nil).Once()
			},
		},
		{
			name:         "assigns role when missing",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor"},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{}, nil).Once()
				rolesRepo.On("GetRoleByName", mock.Anything, "Editor").Return(&types.Role{ID: "role-1", Name: "Editor"}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Editor"}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(nil).Once()
			},
		},
		{
			name:         "role lookup failure is logged and ignored",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor"},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{}, nil).Once()
				rolesRepo.On("GetRoleByName", mock.Anything, "Editor").Return((*types.Role)(nil), errors.New("lookup failed")).Once()
			},
			wantErrors:     1,
			wantErrMessage: "failed to resolve role",
		},
		{
			name:         "assignment failure is logged and ignored",
			contextValue: authmodels.AccessControlAssignRoleContext{UserID: "user-1", RoleName: "Editor"},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{}, nil).Once()
				rolesRepo.On("GetRoleByName", mock.Anything, "Editor").Return(&types.Role{ID: "role-1", Name: "Editor"}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "Editor"}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(errors.New("assign failed")).Once()
			},
			wantErrors:     1,
			wantErrMessage: "failed to assign role",
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

			logger := &hookTestLogger{}
			plugin := newAccessControlHookTestPlugin(logger, rolesRepo, userRolesRepo)

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

			if len(logger.errors) != tc.wantErrors {
				t.Fatalf("expected %d logged errors, got %d: %#v", tc.wantErrors, len(logger.errors), logger.errors)
			}
			if tc.wantErrMessage != "" && (len(logger.errors) == 0 || !containsString(logger.errors[0], tc.wantErrMessage)) {
				t.Fatalf("expected log message containing %q, got %#v", tc.wantErrMessage, logger.errors)
			}

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}

func containsString(value string, substring string) bool {
	return strings.Contains(value, substring)
}
