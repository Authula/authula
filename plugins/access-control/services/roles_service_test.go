package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
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
			wantErr:  coreerrors.ErrBadRequest,
		},
		{
			name:     "not found",
			roleName: "missing",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "missing").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
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
			wantErr: coreerrors.ErrConflict,
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

func TestRolesServiceCreateRoleValidation(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("create failed")

	tests := []struct {
		name      string
		req       types.CreateRoleRequest
		setup     func(*accesscontroltests.MockRolesRepository)
		wantErr   error
		wantCheck func(*testing.T, *types.Role)
	}{
		{
			name:    "blank name",
			req:     types.CreateRoleRequest{Name: ""},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "repository error",
			req:  types.CreateRoleRequest{Name: "editor"},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("CreateRole", mock.Anything, mock.AnythingOfType("*types.Role")).Return(repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name: "explicit weight without description",
			req:  types.CreateRoleRequest{Name: "editor", Weight: new(50), IsSystem: true},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("CreateRole", mock.Anything, mock.MatchedBy(func(role *types.Role) bool {
					return role != nil && role.ID != "" && role.Name == "editor" && role.Description == nil && role.Weight == 50 && role.IsSystem
				})).Return(nil).Once()
			},
			wantCheck: func(t *testing.T, role *types.Role) {
				t.Helper()
				if role == nil || role.Weight != 50 || role.Description != nil || !role.IsSystem {
					t.Fatalf("unexpected role %#v", role)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo)
			}

			service := NewRolesService(rolesRepo, nil, nil)
			role, err := service.CreateRole(context.Background(), tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil && role != nil {
				t.Fatalf("expected nil role, got %#v", role)
			}
			if tc.wantCheck != nil {
				tc.wantCheck(t, role)
			}

			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestRolesServiceGetAllRoles(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("list failed")

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockRolesRepository)
		want    []types.Role
		wantErr error
	}{
		{
			name: "success",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetAllRoles", mock.Anything).Return([]types.Role{{ID: "role-1", Name: "admin"}}, nil).Once()
			},
			want: []types.Role{{ID: "role-1", Name: "admin"}},
		},
		{
			name: "repository error",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetAllRoles", mock.Anything).Return(([]types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			tc.setup(rolesRepo)

			service := NewRolesService(rolesRepo, nil, nil)
			roles, err := service.GetAllRoles(context.Background())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if roles != nil {
					t.Fatalf("expected nil roles, got %#v", roles)
				}
				rolesRepo.AssertExpectations(t)
				return
			}
			if len(roles) != len(tc.want) || roles[0].ID != tc.want[0].ID {
				t.Fatalf("unexpected roles %#v", roles)
			}

			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestRolesServiceGetRoleByIDErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")
	permissionErr := errors.New("permissions failed")

	tests := []struct {
		name    string
		roleID  string
		setup   func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockRolePermissionsRepository)
		wantErr error
	}{
		{
			name:    "blank id",
			roleID:  "",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "repository error",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "role not found",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "role permission error",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				rolePermissionsRepo.On("GetRolePermissions", mock.Anything, "role-1").Return(([]types.UserPermissionInfo)(nil), permissionErr).Once()
			},
			wantErr: permissionErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo, rolePermissionsRepo)
			}

			service := NewRolesService(rolesRepo, rolePermissionsRepo, nil)
			details, err := service.GetRoleByID(context.Background(), tc.roleID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil && details != nil {
				t.Fatalf("expected nil details, got %#v", details)
			}

			rolesRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}

func TestRolesServiceUpdateRoleErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")
	updateErr := errors.New("update failed")
	refetchErr := errors.New("refetch failed")

	tests := []struct {
		name    string
		roleID  string
		req     types.UpdateRoleRequest
		setup   func(*accesscontroltests.MockRolesRepository)
		wantErr error
	}{
		{
			name:    "blank id",
			roleID:  "",
			req:     types.UpdateRoleRequest{Name: new("editor")},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "no fields to update",
			roleID:  "role-1",
			req:     types.UpdateRoleRequest{},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:   "lookup error",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "role not found",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "system role",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", IsSystem: true}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrCannotUpdateSystemRole,
		},
		{
			name:   "blank name",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "update error",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				rolesRepo.On("UpdateRole", mock.Anything, "role-1", new("editor"), (*string)(nil), (*int)(nil)).Return(false, updateErr).Once()
			},
			wantErr: updateErr,
		},
		{
			name:   "update reports no rows",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				rolesRepo.On("UpdateRole", mock.Anything, "role-1", new("editor"), (*string)(nil), (*int)(nil)).Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "refetch error",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				rolesRepo.On("UpdateRole", mock.Anything, "role-1", new("editor"), (*string)(nil), (*int)(nil)).Return(true, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), refetchErr).Once()
			},
			wantErr: refetchErr,
		},
		{
			name:   "refetch returns nil",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				rolesRepo.On("UpdateRole", mock.Anything, "role-1", new("editor"), (*string)(nil), (*int)(nil)).Return(true, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "success with name only",
			roleID: "role-1",
			req:    types.UpdateRoleRequest{Name: new("editor")},
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "viewer"}, nil).Once()
				rolesRepo.On("UpdateRole", mock.Anything, "role-1", new("editor"), (*string)(nil), (*int)(nil)).Return(true, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "editor"}, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo)
			}

			service := NewRolesService(rolesRepo, nil, nil)
			role, err := service.UpdateRole(context.Background(), tc.roleID, tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if role != nil {
					t.Fatalf("expected nil role, got %#v", role)
				}
				rolesRepo.AssertExpectations(t)
				return
			}
			if role == nil || role.Name != "editor" {
				t.Fatalf("unexpected role %#v", role)
			}

			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestRolesServiceDeleteRoleErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")
	countErr := errors.New("count failed")
	deleteErr := errors.New("delete failed")

	tests := []struct {
		name    string
		roleID  string
		setup   func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		wantErr error
	}{
		{
			name:    "blank id",
			roleID:  "",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "lookup error",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "role not found",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "system role",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", IsSystem: true}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrCannotUpdateSystemRole,
		},
		{
			name:   "user count error",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				userRolesRepo.On("CountUsersByRole", mock.Anything, "role-1").Return(0, countErr).Once()
			},
			wantErr: countErr,
		},
		{
			name:   "delete error",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				userRolesRepo.On("CountUsersByRole", mock.Anything, "role-1").Return(0, nil).Once()
				rolesRepo.On("DeleteRole", mock.Anything, "role-1").Return(false, deleteErr).Once()
			},
			wantErr: deleteErr,
		},
		{
			name:   "delete reports no rows",
			roleID: "role-1",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				userRolesRepo.On("CountUsersByRole", mock.Anything, "role-1").Return(0, nil).Once()
				rolesRepo.On("DeleteRole", mock.Anything, "role-1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo, userRolesRepo)
			}

			service := NewRolesService(rolesRepo, nil, userRolesRepo)
			err := service.DeleteRole(context.Background(), tc.roleID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}
