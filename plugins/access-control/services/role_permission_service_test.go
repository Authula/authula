package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
	servicespkg "github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/services"
	testshelpers "github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

func newRolePermissionServiceFixture() (*servicespkg.RolePermissionService, *testshelpers.MockRolePermissionRepository) {
	repo := testshelpers.NewMockRolePermissionRepository()
	svc := servicespkg.NewRolePermissionService(repo)
	return svc, repo
}

func TestRolePermissionServiceCreateRoleTrimsName(t *testing.T) {
	t.Parallel()

	svc, repo := newRolePermissionServiceFixture()
	repo.On("CreateRole", mock.Anything, mock.AnythingOfType("*types.Role")).
		Run(func(args mock.Arguments) {
			role := args.Get(1).(*types.Role)
			if role.Name != "admin" {
				t.Fatalf("expected trimmed role name, got %q", role.Name)
			}
		}).
		Return(nil).
		Once()

	role, err := svc.CreateRole(context.Background(), types.CreateRoleRequest{Name: "  admin  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.Name != "admin" {
		t.Fatalf("expected trimmed role name, got %q", role.Name)
	}
	repo.AssertExpectations(t)
}

func TestRolePermissionServiceAssignRoleToUserRejectsPastExpiry(t *testing.T) {
	t.Parallel()

	svc, _ := newRolePermissionServiceFixture()
	past := time.Now().UTC().Add(-1 * time.Hour)

	err := svc.AssignRoleToUser(
		context.Background(),
		"user-1",
		types.AssignUserRoleRequest{RoleID: "role-1", ExpiresAt: &past},
		nil,
	)
	if !errors.Is(err, constants.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestRolePermissionServiceDeleteRoleReturnsConflictWhenAssigned(t *testing.T) {
	t.Parallel()

	svc, repo := newRolePermissionServiceFixture()
	ctx := context.Background()
	roleID := "role-1"

	repo.On("GetRoleByID", mock.Anything, roleID).
		Return(&types.Role{ID: roleID, Name: "role-for-delete"}, nil).
		Once()
	repo.On("CountUserAssignmentsByRoleID", mock.Anything, roleID).
		Return(1, nil).
		Once()

	err := svc.DeleteRole(ctx, roleID)
	if !errors.Is(err, constants.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	repo.AssertExpectations(t)
}

func TestRolePermissionServiceGetRolePermissionsRejectsEmptyRoleID(t *testing.T) {
	t.Parallel()

	svc, _ := newRolePermissionServiceFixture()

	_, err := svc.GetRolePermissions(context.Background(), "   ")
	if !errors.Is(err, constants.ErrUnprocessableEntity) {
		t.Fatalf("expected ErrUnprocessableEntity, got %v", err)
	}
}

func TestRolePermissionServiceGetRolePermissionsReturnsNotFoundForMissingRole(t *testing.T) {
	t.Parallel()

	svc, repo := newRolePermissionServiceFixture()
	repo.On("GetRoleByID", mock.Anything, "missing-role").Return((*types.Role)(nil), nil).Once()

	_, err := svc.GetRolePermissions(context.Background(), "missing-role")
	if !errors.Is(err, constants.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	repo.AssertExpectations(t)
}

func TestRolePermissionServiceGetRolePermissionsReturnsAssignedPermissions(t *testing.T) {
	t.Parallel()

	svc, repo := newRolePermissionServiceFixture()
	ctx := context.Background()
	roleID := "role-1"
	permissionID := "perm-1"
	permissionKey := "users.read"

	repo.On("GetRoleByID", mock.Anything, roleID).
		Return(&types.Role{ID: roleID, Name: "RolePermReader"}, nil).
		Once()
	repo.On("GetRolePermissions", mock.Anything, roleID).
		Return([]types.UserPermissionInfo{{PermissionID: permissionID, PermissionKey: permissionKey}}, nil).
		Once()

	permissions, err := svc.GetRolePermissions(ctx, "  "+roleID+"  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(permissions))
	}
	if permissions[0].PermissionID != permissionID {
		t.Fatalf("expected permission id %q, got %q", permissionID, permissions[0].PermissionID)
	}
	if permissions[0].PermissionKey != permissionKey {
		t.Fatalf("expected permission key %q, got %q", permissionKey, permissions[0].PermissionKey)
	}
	repo.AssertExpectations(t)
}
