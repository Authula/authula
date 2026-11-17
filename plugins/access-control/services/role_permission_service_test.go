package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

type rolePermissionRepos struct {
	roles           *accesscontroltests.MockRolesRepository
	permissions     *accesscontroltests.MockPermissionsRepository
	rolePermissions *accesscontroltests.MockRolePermissionsRepository
}

func newRolePermissionRepos() rolePermissionRepos {
	return rolePermissionRepos{
		roles:           &accesscontroltests.MockRolesRepository{},
		permissions:     &accesscontroltests.MockPermissionsRepository{},
		rolePermissions: &accesscontroltests.MockRolePermissionsRepository{},
	}
}

func (r rolePermissionRepos) assertExpectations(t *testing.T) {
	t.Helper()
	r.roles.AssertExpectations(t)
	r.permissions.AssertExpectations(t)
	r.rolePermissions.AssertExpectations(t)
}

func TestRolePermissionsServiceGetRolePermissions(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("role repo failed")
	permissionErr := errors.New("role permission repo failed")

	tests := []struct {
		name    string
		roleID  string
		setup   func(rolePermissionRepos)
		want    []types.UserPermissionInfo
		wantErr error
	}{
		{
			name:    "blank role id",
			roleID:  "",
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:   "role lookup error",
			roleID: "role-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "role not found",
			roleID: "role-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "role permission lookup error",
			roleID: "role-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.rolePermissions.On("GetRolePermissions", mock.Anything, "role-1").Return(([]types.UserPermissionInfo)(nil), permissionErr).Once()
			},
			wantErr: permissionErr,
		},
		{
			name:   "success",
			roleID: "role-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.rolePermissions.On("GetRolePermissions", mock.Anything, "role-1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}, nil).Once()
			},
			want: []types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repos := newRolePermissionRepos()
			if tc.setup != nil {
				tc.setup(repos)
			}

			service := NewRolePermissionsService(repos.roles, repos.permissions, repos.rolePermissions)
			got, err := service.GetRolePermissions(context.Background(), tc.roleID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil && got != nil {
				t.Fatalf("expected nil permissions, got %#v", got)
			}
			if tc.wantErr == nil && len(got) != len(tc.want) {
				t.Fatalf("expected %d permissions, got %#v", len(tc.want), got)
			}

			repos.assertExpectations(t)
		})
	}
}

func TestRolePermissionsServiceAddPermissionToRole(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("role repo failed")
	permissionErr := errors.New("permission repo failed")
	grantErr := errors.New("grant failed")
	grantedBy := "u2"

	tests := []struct {
		name         string
		roleID       string
		permissionID string
		setup        func(rolePermissionRepos)
		wantErr      error
	}{
		{
			name:         "blank role id",
			roleID:       "",
			permissionID: "perm-1",
			wantErr:      coreerrors.ErrBadRequest,
		},
		{
			name:         "blank permission id",
			roleID:       "role-1",
			permissionID: "",
			wantErr:      coreerrors.ErrBadRequest,
		},
		{
			name:         "role lookup error",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:         "role not found",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:         "system role",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", IsSystem: true}, nil).Once()
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:         "permission lookup error",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), permissionErr).Once()
			},
			wantErr: permissionErr,
		},
		{
			name:         "permission not found",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:         "grant error",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				repos.rolePermissions.On("AddRolePermission", mock.Anything, "role-1", "perm-1", &grantedBy).Return(grantErr).Once()
			},
			wantErr: grantErr,
		},
		{
			name:         "success",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				repos.rolePermissions.On("AddRolePermission", mock.Anything, "role-1", "perm-1", &grantedBy).Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repos := newRolePermissionRepos()
			if tc.setup != nil {
				tc.setup(repos)
			}

			service := NewRolePermissionsService(repos.roles, repos.permissions, repos.rolePermissions)
			err := service.AddPermissionToRole(context.Background(), tc.roleID, tc.permissionID, &grantedBy)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			repos.assertExpectations(t)
		})
	}
}

func TestRolePermissionsServiceRemovePermissionFromRole(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("role repo failed")
	permissionErr := errors.New("permission repo failed")
	removeErr := errors.New("remove failed")

	tests := []struct {
		name         string
		roleID       string
		permissionID string
		setup        func(rolePermissionRepos)
		wantErr      error
	}{
		{
			name:         "blank role id",
			roleID:       "",
			permissionID: "perm-1",
			wantErr:      coreerrors.ErrUnprocessableEntity,
		},
		{
			name:         "blank permission id",
			roleID:       "role-1",
			permissionID: "",
			wantErr:      coreerrors.ErrUnprocessableEntity,
		},
		{
			name:         "role lookup error",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:         "role not found",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:         "system role",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", IsSystem: true}, nil).Once()
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:         "permission lookup error",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), permissionErr).Once()
			},
			wantErr: permissionErr,
		},
		{
			name:         "permission not found",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:         "remove error",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				repos.rolePermissions.On("RemoveRolePermission", mock.Anything, "role-1", "perm-1").Return(removeErr).Once()
			},
			wantErr: removeErr,
		},
		{
			name:         "success",
			roleID:       "role-1",
			permissionID: "perm-1",
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				repos.rolePermissions.On("RemoveRolePermission", mock.Anything, "role-1", "perm-1").Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repos := newRolePermissionRepos()
			if tc.setup != nil {
				tc.setup(repos)
			}

			service := NewRolePermissionsService(repos.roles, repos.permissions, repos.rolePermissions)
			err := service.RemovePermissionFromRole(context.Background(), tc.roleID, tc.permissionID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			repos.assertExpectations(t)
		})
	}
}

func TestRolePermissionsServiceReplaceRolePermissions(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("role repo failed")
	permissionErr := errors.New("permission repo failed")
	replaceErr := errors.New("replace failed")
	grantedBy := "u2"

	tests := []struct {
		name           string
		roleID         string
		permissionIDs  []string
		setup          func(rolePermissionRepos)
		wantNormalized []string
		wantErr        error
	}{
		{
			name:          "blank role id",
			roleID:        "",
			permissionIDs: []string{"perm-1"},
			wantErr:       coreerrors.ErrBadRequest,
		},
		{
			name:          "role lookup error",
			roleID:        "role-1",
			permissionIDs: []string{"perm-1"},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:          "role not found",
			roleID:        "role-1",
			permissionIDs: []string{"perm-1"},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:          "system role",
			roleID:        "role-1",
			permissionIDs: []string{"perm-1"},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", IsSystem: true}, nil).Once()
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:          "permission lookup error",
			roleID:        "role-1",
			permissionIDs: []string{"perm-1"},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), permissionErr).Once()
			},
			wantErr: permissionErr,
		},
		{
			name:          "permission not found",
			roleID:        "role-1",
			permissionIDs: []string{"perm-1"},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:          "replace error",
			roleID:        "role-1",
			permissionIDs: []string{"perm-1"},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				repos.rolePermissions.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-1"}, &grantedBy).Return(replaceErr).Once()
			},
			wantErr: replaceErr,
		},
		{
			name:          "deduplicates and drops blank permission ids",
			roleID:        "role-1",
			permissionIDs: []string{"", "perm-1", "perm-1", "perm-2"},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				repos.permissions.On("GetPermissionByID", mock.Anything, "perm-2").Return(&types.Permission{ID: "perm-2"}, nil).Once()
				repos.rolePermissions.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{"perm-1", "perm-2"}, &grantedBy).Return(nil).Once()
			},
			wantNormalized: []string{"perm-1", "perm-2"},
		},
		{
			name:          "empty list replaces with no permissions",
			roleID:        "role-1",
			permissionIDs: []string{},
			setup: func(repos rolePermissionRepos) {
				repos.roles.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1"}, nil).Once()
				repos.rolePermissions.On("ReplaceRolePermissions", mock.Anything, "role-1", []string{}, &grantedBy).Return(nil).Once()
			},
			wantNormalized: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repos := newRolePermissionRepos()
			if tc.setup != nil {
				tc.setup(repos)
			}

			service := NewRolePermissionsService(repos.roles, repos.permissions, repos.rolePermissions)
			err := service.ReplaceRolePermissions(context.Background(), tc.roleID, tc.permissionIDs, &grantedBy)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantNormalized != nil && len(tc.wantNormalized) == 0 {
				repos.rolePermissions.AssertCalled(t, "ReplaceRolePermissions", mock.Anything, "role-1", []string{}, &grantedBy)
			}

			repos.assertExpectations(t)
		})
	}
}
