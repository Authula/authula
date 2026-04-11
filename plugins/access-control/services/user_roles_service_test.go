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

func TestUserRolesServiceAssignRoleToUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		req              types.AssignUserRoleRequest
		assignedByUserID *string
		setup            func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockRolesRepository)
		wantErr          error
	}{
		{
			name:    "expired assignment",
			req:     types.AssignUserRoleRequest{RoleID: "role-1", ExpiresAt: func() *time.Time { t := time.Now().UTC().Add(-time.Hour); return &t }()},
			wantErr: accesscontrolconstants.ErrBadRequest,
		},
		{
			name: "success without assigner",
			req:  types.AssignUserRoleRequest{RoleID: "role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin", Weight: 20}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(nil).Once()
			},
		},
		{
			name:             "success with sufficient assigner weight",
			req:              types.AssignUserRoleRequest{RoleID: "role-1"},
			assignedByUserID: func() *string { v := "assigner-1"; return &v }(),
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin", Weight: 20}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-2", RoleName: "owner", RoleWeight: 50}}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", mock.MatchedBy(func(userID *string) bool {
					return userID != nil && *userID == "assigner-1"
				}), (*time.Time)(nil)).Return(nil).Once()
			},
		},
		{
			name:             "forbidden when target role is higher than assigner",
			req:              types.AssignUserRoleRequest{RoleID: "role-1"},
			assignedByUserID: func() *string { v := "assigner-1"; return &v }(),
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin", Weight: 80}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-2", RoleName: "member", RoleWeight: 10}}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setup != nil {
				tc.setup(userRolesRepo, rolesRepo)
			}

			service := NewUserRolesService(userRolesRepo, rolesRepo)
			err := service.AssignRoleToUser(context.Background(), "user-1", tc.req, tc.assignedByUserID)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			userRolesRepo.AssertExpectations(t)
			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestUserRolesServiceReplaceUserRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		roleIDs        []string
		assignerUserID *string
		setup          func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockRolesRepository)
		wantErr        error
	}{
		{
			name:    "dedupes role ids",
			roleIDs: []string{"role-1", "role-1", "role-2"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin", Weight: 20}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-2").Return(&types.Role{ID: "role-2", Name: "editor", Weight: 10}, nil).Once()
				userRolesRepo.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-1", "role-2"}, (*string)(nil)).Return(nil).Once()
			},
		},
		{
			name:           "forbidden when assigner lacks sufficient weight",
			roleIDs:        []string{"role-1", "role-2"},
			assignerUserID: func() *string { value := "assigner-1"; return &value }(),
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin", Weight: 20}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-2").Return(&types.Role{ID: "role-2", Name: "editor", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-10", RoleName: "manager", RoleWeight: 15}}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setup != nil {
				tc.setup(userRolesRepo, rolesRepo)
			}

			service := NewUserRolesService(userRolesRepo, rolesRepo)
			err := service.ReplaceUserRoles(context.Background(), "user-1", tc.roleIDs, tc.assignerUserID)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}
