package services

import (
	"context"
	"testing"
	"time"

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

func TestRolesServiceRoleWeightOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockRolePermissionsRepository, *accesscontroltests.MockUserRolesRepository)
		run      func(context.Context, *RolesService) (*types.Role, error)
		wantRole func(*types.Role) bool
	}{
		{
			name: "create defaults weight",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				description := "default weighted role"
				fixedTime := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
				rolesRepo.On("CreateRole", mock.Anything, mock.MatchedBy(func(role *types.Role) bool {
					return role != nil && role.Name == "editor" && role.Description != nil && *role.Description == description && role.Weight == 10
				})).Run(func(args mock.Arguments) {
					role := args.Get(1).(*types.Role)
					role.ID = "role-1"
					role.CreatedAt = fixedTime
					role.UpdatedAt = fixedTime
				}).Return(nil).Once()
			},
			run: func(ctx context.Context, service *RolesService) (*types.Role, error) {
				return service.CreateRole(ctx, types.CreateRoleRequest{Name: "editor", Description: func() *string { s := "default weighted role"; return &s }(), IsSystem: false})
			},
			wantRole: func(role *types.Role) bool { return role != nil && role.Weight == 10 && role.ID == "role-1" },
		},
		{
			name: "update supports weight",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				description := new(string)
				*description = "updated role"
				updatedDescription := new(string)
				*updatedDescription = "updated role description"
				newWeight := 40
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor", Description: description, Weight: 10}, nil).Once()
				rolesRepo.On("UpdateRole", mock.Anything, "role-1", (*string)(nil), updatedDescription, &newWeight).Return(true, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor", Description: updatedDescription, Weight: newWeight}, nil).Once()
			},
			run: func(ctx context.Context, service *RolesService) (*types.Role, error) {
				updatedDescription := "updated role description"
				weight := 40
				return service.UpdateRole(ctx, "role-1", types.UpdateRoleRequest{Description: &updatedDescription, Weight: &weight})
			},
			wantRole: func(role *types.Role) bool {
				return role != nil && role.Weight == 40 && role.Description != nil && *role.Description == "updated role description"
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
				tc.setup(rolesRepo, rolePermissionsRepo, userRolesRepo)
			}

			service := NewRolesService(rolesRepo, rolePermissionsRepo, userRolesRepo)
			role, err := tc.run(context.Background(), service)
			if err != nil {
				t.Fatalf("expected nil err, got %v", err)
			}
			if tc.wantRole != nil && !tc.wantRole(role) {
				t.Fatalf("unexpected role %#v", role)
			}

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
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
