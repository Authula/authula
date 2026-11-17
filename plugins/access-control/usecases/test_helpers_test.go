package usecases

import (
	"reflect"
	"testing"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/services"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

type useCaseMocks struct {
	roles           *accesscontroltests.MockRolesRepository
	permissions     *accesscontroltests.MockPermissionsRepository
	rolePermissions *accesscontroltests.MockRolePermissionsRepository
	userRoles       *accesscontroltests.MockUserRolesRepository
	userPermissions *accesscontroltests.MockUserPermissionsRepository
	authorizer      *internaltests.MockAuthorizer
}

func newUseCaseMocks() *useCaseMocks {
	return &useCaseMocks{
		roles:           &accesscontroltests.MockRolesRepository{},
		permissions:     &accesscontroltests.MockPermissionsRepository{},
		rolePermissions: &accesscontroltests.MockRolePermissionsRepository{},
		userRoles:       &accesscontroltests.MockUserRolesRepository{},
		userPermissions: &accesscontroltests.MockUserPermissionsRepository{},
		authorizer:      &internaltests.MockAuthorizer{},
	}
}

func (m *useCaseMocks) rolesUseCase() *RolesUseCase {
	return NewRolesUseCase(services.NewRolesService(m.roles, m.rolePermissions, m.userRoles), m.authorizer)
}

func (m *useCaseMocks) permissionsUseCase() *PermissionsUseCase {
	return NewPermissionsUseCase(services.NewPermissionsService(m.permissions, m.rolePermissions), m.authorizer)
}

func (m *useCaseMocks) rolePermissionsUseCase() *RolePermissionsUseCase {
	return NewRolePermissionsUseCase(services.NewRolePermissionsService(m.roles, m.permissions, m.rolePermissions), m.authorizer)
}

func (m *useCaseMocks) userRolesUseCase() *UserRolesUseCase {
	return NewUserRolesUseCase(services.NewUserRolesService(m.userRoles, m.roles), m.authorizer)
}

func (m *useCaseMocks) userPermissionsUseCase() *UserPermissionsUseCase {
	return NewUserPermissionsUseCase(services.NewUserPermissionsService(m.userPermissions), m.authorizer)
}

func (m *useCaseMocks) assertExpectations(t *testing.T) {
	t.Helper()
	m.roles.AssertExpectations(t)
	m.permissions.AssertExpectations(t)
	m.rolePermissions.AssertExpectations(t)
	m.userRoles.AssertExpectations(t)
	m.userPermissions.AssertExpectations(t)
	m.authorizer.AssertExpectations(t)
}

func testActor() *models.Actor {
	return &models.Actor{ID: "actor-1", Type: models.ActorUser}
}

func isNilResult(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func assertRole(t *testing.T, result any, wantID string) {
	t.Helper()
	role, ok := result.(*types.Role)
	if !ok || role == nil || role.ID != wantID {
		t.Fatalf("expected role %q, got %#v", wantID, result)
	}
}

func assertPermissions(t *testing.T, result any, want int) {
	t.Helper()
	permissions, ok := result.([]types.Permission)
	if !ok || len(permissions) != want {
		t.Fatalf("expected %d permissions, got %#v", want, result)
	}
}

func assertRolePermissions(t *testing.T, result any, want int) {
	t.Helper()
	permissions, ok := result.([]types.UserPermissionInfo)
	if !ok || len(permissions) != want {
		t.Fatalf("expected %d role permissions, got %#v", want, result)
	}
}

func assertUserRoles(t *testing.T, result any, want int) {
	t.Helper()
	roles, ok := result.([]types.UserRoleInfo)
	if !ok || len(roles) != want {
		t.Fatalf("expected %d user roles, got %#v", want, result)
	}
}

func assertBool(t *testing.T, result any, want bool) {
	t.Helper()
	value, ok := result.(bool)
	if !ok || value != want {
		t.Fatalf("expected %v, got %#v", want, result)
	}
}
