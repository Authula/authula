package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestRolesServiceGetRoleByID(t *testing.T) {
	t.Parallel()

	rolesRepo := &accesscontroltests.MockRolesRepository{}
	rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
	userAccessRepo := &accesscontroltests.MockUserAccessRepository{}

	rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
	rolePermissionsRepo.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}, nil).Once()

	service := NewRolesService(rolesRepo, rolePermissionsRepo, userAccessRepo)
	details, err := service.GetRoleByID(context.Background(), "role-1")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if details == nil || details.Role.ID != "role-1" || len(details.Permissions) != 1 {
		t.Fatalf("unexpected details %#v", details)
	}

	rolesRepo.AssertExpectations(t)
	rolePermissionsRepo.AssertExpectations(t)
}

func TestRolesServiceDeleteRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserAccessRepository)
		wantErr error
	}{
		{
			name: "role in use",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
				userAccessRepo.On("CountUserAssignmentsByRoleID", mock.Anything, "role-1").Return(1, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrConflict,
		},
		{
			name: "success",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
				userAccessRepo.On("CountUserAssignmentsByRoleID", mock.Anything, "role-1").Return(0, nil).Once()
				rolesRepo.On("DeleteRole", mock.Anything, "role-1").Return(true, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo, userAccessRepo)
			}

			service := NewRolesService(rolesRepo, rolePermissionsRepo, userAccessRepo)
			err := service.DeleteRole(context.Background(), "role-1")
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			rolesRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}
