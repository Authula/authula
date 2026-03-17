package usecases

import (
	"context"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type stubUserAccessRepo struct {
	permissions []types.UserPermissionInfo
}

func (s *stubUserAccessRepo) GetUserRoles(_ context.Context, _ string) ([]types.UserRoleInfo, error) {
	return []types.UserRoleInfo{{RoleID: "role-1", RoleName: "admin"}}, nil
}

func (s *stubUserAccessRepo) GetUserEffectivePermissions(_ context.Context, _ string) ([]types.UserPermissionInfo, error) {
	return s.permissions, nil
}

func (s *stubUserAccessRepo) GetUserWithRolesByID(_ context.Context, _ string) (*types.UserWithRoles, error) {
	return &types.UserWithRoles{}, nil
}

func (s *stubUserAccessRepo) GetUserWithPermissionsByID(_ context.Context, _ string) (*types.UserWithPermissions, error) {
	return &types.UserWithPermissions{}, nil
}

func TestUserRolesUseCaseHasPermissionsPassThrough(t *testing.T) {
	repo := &stubUserAccessRepo{permissions: []types.UserPermissionInfo{{PermissionKey: "users.read"}}}
	uc := NewUserRolesUseCase(services.NewUserAccessService(repo))

	ok, err := uc.HasPermissions(context.Background(), "user-1", []string{"users.read"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected permission check to pass")
	}
}
