package services

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestAccessControlServiceRoleExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		roleName string
		setup    func(*accesscontroltests.MockRolesRepository)
		wantErr  error
		wantOK   bool
	}{
		{
			name:     "role not found",
			roleName: "missing",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "missing").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
			wantOK:  false,
		},
		{
			name:     "role exists",
			roleName: "editor",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
			},
			wantOK: true,
		},
		{
			name:     "repository error",
			roleName: "editor",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return((*types.Role)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo)
			}

			service := NewAccessControlService(NewRolesService(rolesRepo, nil, nil), NewUserRolesService(nil, nil), nil)
			ok, err := service.RoleExists(context.Background(), tc.roleName)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if ok {
					t.Fatalf("expected false, got true")
				}
			} else if ok != tc.wantOK {
				t.Fatalf("unexpected result %v", ok)
			}

			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestAccessControlServiceValidatePermissionKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		permissionKeys []string
		setup          func(*accesscontroltests.MockPermissionsRepository)
		wantErr        error
	}{
		{
			name:           "empty keys returns nil",
			permissionKeys: []string{},
			wantErr:        nil,
		},
		{
			name:           "all keys exist",
			permissionKeys: []string{"read:users", "write:users"},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "read:users").Return(&types.Permission{ID: "p1", Key: "read:users"}, nil).Once()
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "write:users").Return(&types.Permission{ID: "p2", Key: "write:users"}, nil).Once()
			},
			wantErr: nil,
		},
		{
			name:           "single missing key returns ErrNotFound",
			permissionKeys: []string{"read:users", "missing:key"},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "read:users").Return(&types.Permission{ID: "p1", Key: "read:users"}, nil).Once()
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "missing:key").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:           "repository error propagates",
			permissionKeys: []string{"read:users"},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "read:users").Return((*types.Permission)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewAccessControlService(
				NewRolesService(nil, nil, nil),
				NewUserRolesService(nil, nil),
				permissionsRepo,
			)
			err := service.ValidatePermissionKeys(context.Background(), tc.permissionKeys)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
		})
	}
}

func TestAccessControlServiceGetRolePermissionsByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		roleName  string
		setup     func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockRolePermissionsRepository)
		wantErr   error
		wantPerms []string
	}{
		{
			name:     "role not found",
			roleName: "missing",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "missing").Return((*types.Role)(nil), coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:     "nil role treated as not found",
			roleName: "ghost",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "ghost").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:     "role permissions resolved",
			roleName: "editor",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
				rolePermRepo.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{
					{PermissionKey: "organizations:members:list"},
					{PermissionKey: "organizations:members:read"},
				}, nil).Once()
			},
			wantPerms: []string{"organizations:members:list", "organizations:members:read"},
		},
		{
			name:     "repository error propagates",
			roleName: "editor",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
				rolePermRepo.On("GetRolePermissions", mock.Anything, "role-1").Return(([]types.UserPermissionInfo)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo, rolePermRepo)
			}

			service := NewAccessControlService(NewRolesService(rolesRepo, rolePermRepo, nil), NewUserRolesService(nil, nil), nil)
			perms, err := service.GetRolePermissionsByName(context.Background(), tc.roleName)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if perms != nil {
					t.Fatalf("expected nil permissions, got %v", perms)
				}
			} else if !slices.Equal(perms, tc.wantPerms) {
				t.Fatalf("expected permissions %v, got %v", tc.wantPerms, perms)
			}

			rolesRepo.AssertExpectations(t)
			rolePermRepo.AssertExpectations(t)
		})
	}
}

func TestAccessControlServiceGetRoleWeightByName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		roleName   string
		setup      func(*accesscontroltests.MockRolesRepository)
		wantErr    error
		wantWeight int
	}{
		{
			name:     "role not found",
			roleName: "missing",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "missing").Return((*types.Role)(nil), coreerrors.ErrNotFound).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:     "nil role treated as not found",
			roleName: "ghost",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "ghost").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:     "role weight resolved",
			roleName: "admin",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "admin").Return(&types.Role{ID: "role-2", Name: "admin", Weight: 80}, nil).Once()
			},
			wantWeight: 80,
		},
		{
			name:     "repository error propagates",
			roleName: "admin",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "admin").Return((*types.Role)(nil), coreerrors.ErrForbidden).Once()
			},
			wantErr: coreerrors.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo)
			}

			service := NewAccessControlService(NewRolesService(rolesRepo, nil, nil), NewUserRolesService(nil, nil), nil)
			weight, err := service.GetRoleWeightByName(context.Background(), tc.roleName)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if weight != 0 {
					t.Fatalf("expected 0 weight, got %d", weight)
				}
			} else if weight != tc.wantWeight {
				t.Fatalf("expected weight %d, got %d", tc.wantWeight, weight)
			}

			rolesRepo.AssertExpectations(t)
		})
	}
}
