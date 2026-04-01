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
		name    string
		req     types.AssignUserRoleRequest
		setup   func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockRolesRepository)
		wantErr error
	}{
		{
			name:    "expired assignment",
			req:     types.AssignUserRoleRequest{RoleID: "role-1", ExpiresAt: func() *time.Time { t := time.Now().UTC().Add(-time.Hour); return &t }()},
			wantErr: accesscontrolconstants.ErrBadRequest,
		},
		{
			name: "success",
			req:  types.AssignUserRoleRequest{RoleID: "role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", (*string)(nil), (*time.Time)(nil)).Return(nil).Once()
			},
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
			err := service.AssignRoleToUser(context.Background(), "user-1", tc.req, nil)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			userRolesRepo.AssertExpectations(t)
			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestUserRolesServiceReplaceUserRolesDedupes(t *testing.T) {
	t.Parallel()

	userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
	rolesRepo := &accesscontroltests.MockRolesRepository{}
	rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin"}, nil).Once()
	rolesRepo.On("GetRoleByID", mock.Anything, "role-2").Return(&types.Role{ID: "role-2", Name: "editor"}, nil).Once()
	userRolesRepo.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-1", "role-2"}, (*string)(nil)).Return(nil).Once()

	service := NewUserRolesService(userRolesRepo, rolesRepo)
	err := service.ReplaceUserRoles(context.Background(), "user-1", []string{"role-1", "role-1", "role-2"}, nil)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}

	rolesRepo.AssertExpectations(t)
	userRolesRepo.AssertExpectations(t)
}
