package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestUserAccessServiceHasPermissions(t *testing.T) {
	t.Parallel()

	userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
	userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
	userAccessRepo.On("GetUserEffectivePermissions", mock.Anything, "user-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}, nil).Once()

	service := NewUserAccessService(userRolesRepo, userAccessRepo)
	ok, err := service.HasPermissions(context.Background(), "user-1", []string{"users.write", "users.read"})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if !ok {
		t.Fatal("expected permission check to pass")
	}

	userAccessRepo.AssertExpectations(t)
}

func TestUserAccessServiceGetUserAuthorizationProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockUserAccessRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "blank user id",
			userID:  "",
			wantErr: accesscontrolconstants.ErrUnprocessableEntity,
		},
		{
			name:   "composes profile",
			userID: "user-1",
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				userRolesRepo.On("GetUserWithRolesByID", mock.Anything, "user-1").Return(&types.UserWithRoles{Roles: []types.UserRoleInfo{{RoleID: "role-1", RoleName: "admin"}}}, nil).Once()
				userAccessRepo.On("GetUserWithPermissionsByID", mock.Anything, "user-1").Return(&types.UserWithPermissions{Permissions: []types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}}, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setup != nil {
				tc.setup(userRolesRepo, userAccessRepo)
			}

			service := NewUserAccessService(userRolesRepo, userAccessRepo)
			profile, err := service.GetUserAuthorizationProfile(context.Background(), tc.userID)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr == nil && profile == nil {
				t.Fatal("expected profile, got nil")
			}

			userRolesRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}
