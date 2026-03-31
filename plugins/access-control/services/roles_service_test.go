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
	userRolesRepo := &accesscontroltests.MockUserRolesRepository{}

	rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
	rolePermissionsRepo.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}, nil).Once()

	service := NewRolesService(rolesRepo, rolePermissionsRepo, userRolesRepo)
	details, err := service.GetRoleByID(context.Background(), "role-1")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if details == nil || details.Role.ID != "role-1" || len(details.Permissions) != 1 {
		t.Fatalf("unexpected details %#v", details)
	}

	rolesRepo.AssertExpectations(t)
	rolePermissionsRepo.AssertExpectations(t)
	userRolesRepo.AssertExpectations(t)
}

func TestRolesServiceGetRoleByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		roleName string
		setup    func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		wantErr  error
	}{
		{
			name:     "bad request",
			roleName: "",
			wantErr:  accesscontrolconstants.ErrBadRequest,
		},
		{
			name:     "not found",
			roleName: "missing",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "missing").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: accesscontrolconstants.ErrNotFound,
		},
		{
			name:     "success",
			roleName: "admin",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "admin").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo, userRolesRepo)
			}

			service := NewRolesService(rolesRepo, rolePermissionsRepo, userRolesRepo)
			role, err := service.GetRoleByName(context.Background(), tc.roleName)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if role != nil {
					t.Fatalf("expected nil role, got %#v", role)
				}
			} else {
				if role == nil || role.ID != "role-1" || role.Name != "admin" {
					t.Fatalf("unexpected role %#v", role)
				}
			}

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}

func TestRolesServiceDeleteRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		wantErr error
	}{
		{
			name: "role in use",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
				userRolesRepo.On("CountUsersByRole", mock.Anything, "role-1").Return(1, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrConflict,
		},
		{
			name: "success",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
				userRolesRepo.On("CountUsersByRole", mock.Anything, "role-1").Return(0, nil).Once()
				rolesRepo.On("DeleteRole", mock.Anything, "role-1").Return(true, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo, userRolesRepo)
			}

			service := NewRolesService(rolesRepo, rolePermissionsRepo, userRolesRepo)
			err := service.DeleteRole(context.Background(), "role-1")
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}
