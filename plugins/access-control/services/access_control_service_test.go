package services

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
	rootservices "github.com/Authula/authula/services"
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

func TestAccessControlServiceAssignRoleToUserIfMissing(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("role lookup failed")
	userRolesErr := errors.New("user roles failed")
	assignErr := errors.New("assign failed")

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		wantErr error
	}{
		{
			name: "role lookup error",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return((*types.Role)(nil), lookupErr).Once()
			},
			wantErr: lookupErr,
		},
		{
			name: "user roles lookup error",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return(([]types.UserRoleInfo)(nil), userRolesErr).Once()
			},
			wantErr: userRolesErr,
		},
		{
			name: "role already assigned",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-1", RoleName: "editor"}}, nil).Once()
			},
		},
		{
			name: "assignment error",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(assignErr).Once()
			},
			wantErr: assignErr,
		},
		{
			name: "assigns missing role",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-2", RoleName: "viewer"}}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			tc.setup(rolesRepo, userRolesRepo)

			service := NewAccessControlService(
				NewRolesService(rolesRepo, nil, nil),
				NewUserRolesService(userRolesRepo, rolesRepo),
				nil,
			)
			err := service.AssignRoleToUserIfMissing(context.Background(), "user-1", "editor", nil)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}

func TestAccessControlServiceEnsurePermissions(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("lookup failed")
	createErr := errors.New("create failed")

	tests := []struct {
		name        string
		definitions []rootservices.PermissionDefinition
		setup       func(*accesscontroltests.MockPermissionsRepository)
		wantErr     error
	}{
		{
			name:        "empty definitions",
			definitions: nil,
		},
		{
			name:        "lookup error",
			definitions: []rootservices.PermissionDefinition{{Key: "users.read", Description: "Read users"}},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "users.read").Return((*types.Permission)(nil), lookupErr).Once()
			},
			wantErr: lookupErr,
		},
		{
			name:        "create error",
			definitions: []rootservices.PermissionDefinition{{Key: "users.read", Description: "Read users"}},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "users.read").Return((*types.Permission)(nil), nil).Once()
				permissionsRepo.On("CreatePermission", mock.Anything, mock.MatchedBy(func(permission *types.Permission) bool {
					return permission != nil && permission.ID != "" && permission.Key == "users.read" && permission.IsSystem && permission.Description != nil && *permission.Description == "Read users"
				})).Return(createErr).Once()
			},
			wantErr: createErr,
		},
		{
			name: "skips existing and creates missing",
			definitions: []rootservices.PermissionDefinition{
				{Key: "users.read", Description: "Read users"},
				{Key: "users.write", Description: "Write users"},
			},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "users.read").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "users.write").Return((*types.Permission)(nil), nil).Once()
				permissionsRepo.On("CreatePermission", mock.Anything, mock.MatchedBy(func(permission *types.Permission) bool {
					return permission != nil && permission.Key == "users.write" && permission.IsSystem
				})).Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewAccessControlService(nil, nil, permissionsRepo)
			err := service.EnsurePermissions(context.Background(), tc.definitions)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
		})
	}
}

func TestAccessControlServiceGetRoleDetailsErrors(t *testing.T) {
	t.Parallel()

	roleErr := errors.New("role lookup failed")

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockRolePermissionsRepository)
		run     func(context.Context, *AccessControlService) error
		wantErr error
	}{
		{
			name: "role permissions detail lookup error",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), roleErr).Once()
			},
			run: func(ctx context.Context, service *AccessControlService) error {
				_, err := service.GetRolePermissionsByName(ctx, "editor")
				return err
			},
			wantErr: roleErr,
		},
		{
			name: "role weight with blank id is not found",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "", Name: "editor"}, nil).Once()
			},
			run: func(ctx context.Context, service *AccessControlService) error {
				_, err := service.GetRoleWeightByName(ctx, "editor")
				return err
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermRepo := &accesscontroltests.MockRolePermissionsRepository{}
			tc.setup(rolesRepo, rolePermRepo)

			service := NewAccessControlService(NewRolesService(rolesRepo, rolePermRepo, nil), NewUserRolesService(nil, nil), nil)
			err := tc.run(context.Background(), service)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			rolesRepo.AssertExpectations(t)
			rolePermRepo.AssertExpectations(t)
		})
	}
}
