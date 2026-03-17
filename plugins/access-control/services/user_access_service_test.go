package services

import (
	"context"
	"errors"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/access-control/types"
)

type stubUserAccessRepo struct {
	rolesResult       []types.UserRoleInfo
	rolesErr          error
	permissionsResult []types.UserPermissionInfo
	permissionsErr    error
	withRolesResult   *types.UserWithRoles
	withRolesErr      error
	withPermsResult   *types.UserWithPermissions
	withPermsErr      error
}

func (s *stubUserAccessRepo) GetUserRoles(_ context.Context, _ string) ([]types.UserRoleInfo, error) {
	return s.rolesResult, s.rolesErr
}

func (s *stubUserAccessRepo) GetUserEffectivePermissions(_ context.Context, _ string) ([]types.UserPermissionInfo, error) {
	return s.permissionsResult, s.permissionsErr
}

func (s *stubUserAccessRepo) GetUserWithRolesByID(_ context.Context, _ string) (*types.UserWithRoles, error) {
	return s.withRolesResult, s.withRolesErr
}

func (s *stubUserAccessRepo) GetUserWithPermissionsByID(_ context.Context, _ string) (*types.UserWithPermissions, error) {
	return s.withPermsResult, s.withPermsErr
}

func TestUserAccessServiceGetUserRolesUnprocessableEntity(t *testing.T) {
	t.Parallel()

	svc := NewUserAccessService(&stubUserAccessRepo{})
	_, err := svc.GetUserRoles(context.Background(), "   ")
	if !errors.Is(err, constants.ErrUnprocessableEntity) {
		t.Fatalf("expected ErrUnprocessableEntity, got %v", err)
	}
}

func TestUserAccessServiceHasPermissionsMatchesAnyRequiredPermission(t *testing.T) {
	t.Parallel()

	svc := NewUserAccessService(&stubUserAccessRepo{
		permissionsResult: []types.UserPermissionInfo{{PermissionKey: "users.read"}, {PermissionKey: "users.write"}},
	})

	ok, err := svc.HasPermissions(context.Background(), "user-1", []string{"billing.read", "users.write"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected permission check to pass when any required permission matches")
	}
}

func TestUserAccessServiceGetUserAuthorizationProfileNilUser(t *testing.T) {
	t.Parallel()

	svc := NewUserAccessService(&stubUserAccessRepo{withRolesResult: nil})

	profile, err := svc.GetUserAuthorizationProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile != nil {
		t.Fatalf("expected nil profile, got %+v", profile)
	}
}
