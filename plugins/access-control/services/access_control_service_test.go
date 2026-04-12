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
			wantErr: accesscontrolconstants.ErrNotFound,
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
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return((*types.Role)(nil), accesscontrolconstants.ErrForbidden).Once()
			},
			wantErr: accesscontrolconstants.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rolesRepo := &accesscontroltests.MockRolesRepository{}
			if tc.setup != nil {
				tc.setup(rolesRepo)
			}

			service := NewAccessControlService(NewRolesService(rolesRepo, nil, nil), NewUserRolesService(nil, nil))
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

func TestAccessControlServiceValidateRoleAssignment(t *testing.T) {
	t.Parallel()

	assignerUserID := func() *string { value := "assigner-user-1"; return &value }()

	tests := []struct {
		name     string
		roleName string
		assigner *string
		setup    func(*accesscontroltests.MockRolesRepository, *accesscontroltests.MockUserRolesRepository)
		wantErr  error
		wantOK   bool
	}{
		{
			name:     "role not found",
			roleName: "missing",
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "missing").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: accesscontrolconstants.ErrNotFound,
		},
		{
			name:     "nil assigner is not allowed",
			roleName: "editor",
			assigner: nil,
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
			},
			wantOK: false,
		},
		{
			name:     "forbidden when assigner has no active roles",
			roleName: "editor",
			assigner: assignerUserID,
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-user-1").Return([]types.UserRoleInfo{{RoleID: "role-old", RoleName: "old", RoleWeight: 100, ExpiresAt: func() *time.Time { value := time.Now().UTC().Add(-time.Hour); return &value }()}}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrForbidden,
		},
		{
			name:     "expired roles are ignored",
			roleName: "editor",
			assigner: assignerUserID,
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "editor").Return(&types.Role{ID: "role-1", Name: "editor", Weight: 20}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-user-1").Return([]types.UserRoleInfo{
					{RoleID: "role-expired", RoleName: "expired", RoleWeight: 100, ExpiresAt: func() *time.Time { value := time.Now().UTC().Add(-time.Hour); return &value }()},
					{RoleID: "role-active", RoleName: "active", RoleWeight: 30},
				}, nil).Once()
			},
			wantOK: true,
		},
		{
			name:     "forbidden when target exceeds assigner weight",
			roleName: "admin",
			assigner: assignerUserID,
			setup: func(rolesRepo *accesscontroltests.MockRolesRepository, userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				rolesRepo.On("GetRoleByName", mock.Anything, "admin").Return(&types.Role{ID: "role-2", Name: "admin", Weight: 80}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-user-1").Return([]types.UserRoleInfo{{RoleID: "role-member", RoleName: "member", RoleWeight: 10}}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrForbidden,
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

			service := NewAccessControlService(NewRolesService(rolesRepo, nil, userRolesRepo), NewUserRolesService(userRolesRepo, rolesRepo))
			ok, err := service.ValidateRoleAssignment(context.Background(), tc.roleName, tc.assigner)
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
			userRolesRepo.AssertExpectations(t)
		})
	}
}
