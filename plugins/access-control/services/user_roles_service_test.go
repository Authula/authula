package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
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
			wantErr: coreerrors.ErrBadRequest,
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
			wantErr: coreerrors.ErrForbidden,
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
		name             string
		roleIDs          []string
		assignedByUserID *string
		setup            func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockRolesRepository)
		wantErr          error
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
			name:             "forbidden when assigner lacks sufficient weight",
			roleIDs:          []string{"role-1", "role-2"},
			assignedByUserID: func() *string { value := "assigner-1"; return &value }(),
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Name: "admin", Weight: 20}, nil).Once()
				rolesRepo.On("GetRoleByID", mock.Anything, "role-2").Return(&types.Role{ID: "role-2", Name: "editor", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-10", RoleName: "manager", RoleWeight: 15}}, nil).Once()
			},
			wantErr: coreerrors.ErrForbidden,
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
			err := service.ReplaceUserRoles(context.Background(), "user-1", tc.roleIDs, tc.assignedByUserID)
			if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			rolesRepo.AssertExpectations(t)
			userRolesRepo.AssertExpectations(t)
		})
	}
}

func TestUserRolesServiceGetUserRoles(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")

	tests := []struct {
		name    string
		userID  string
		setup   func(*accesscontroltests.MockUserRolesRepository)
		want    []types.UserRoleInfo
		wantErr error
	}{
		{
			name:    "blank user id",
			userID:  "",
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:   "repository error",
			userID: "user-1",
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return(([]types.UserRoleInfo)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "success",
			userID: "user-1",
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("GetUserRoles", mock.Anything, "user-1").Return([]types.UserRoleInfo{{RoleID: "role-1", RoleName: "editor"}}, nil).Once()
			},
			want: []types.UserRoleInfo{{RoleID: "role-1", RoleName: "editor"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setup != nil {
				tc.setup(userRolesRepo)
			}

			service := NewUserRolesService(userRolesRepo, nil)
			roles, err := service.GetUserRoles(context.Background(), tc.userID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if roles != nil {
					t.Fatalf("expected nil roles, got %#v", roles)
				}
				userRolesRepo.AssertExpectations(t)
				return
			}
			if len(roles) != len(tc.want) || roles[0].RoleID != tc.want[0].RoleID {
				t.Fatalf("unexpected roles %#v", roles)
			}

			userRolesRepo.AssertExpectations(t)
		})
	}
}

func TestUserRolesServiceRemoveRoleFromUser(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("remove failed")

	tests := []struct {
		name    string
		userID  string
		roleID  string
		setup   func(*accesscontroltests.MockUserRolesRepository)
		wantErr error
	}{
		{
			name:    "blank user id",
			userID:  "",
			roleID:  "role-1",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "blank role id",
			userID:  "user-1",
			roleID:  "",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "repository error",
			userID: "user-1",
			roleID: "role-1",
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "success",
			userID: "user-1",
			roleID: "role-1",
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository) {
				userRolesRepo.On("RemoveUserRole", mock.Anything, "user-1", "role-1").Return(nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRolesRepo := &accesscontroltests.MockUserRolesRepository{}
			if tc.setup != nil {
				tc.setup(userRolesRepo)
			}

			service := NewUserRolesService(userRolesRepo, nil)
			err := service.RemoveRoleFromUser(context.Background(), tc.userID, tc.roleID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			userRolesRepo.AssertExpectations(t)
		})
	}
}

func TestUserRolesServiceReplaceUserRolesErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")
	assignerErr := errors.New("assigner lookup failed")
	replaceErr := errors.New("replace failed")
	assignedBy := "assigner-1"

	tests := []struct {
		name    string
		userID  string
		roleIDs []string
		setup   func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockRolesRepository)
		wantErr error
	}{
		{
			name:    "blank user id",
			userID:  "",
			roleIDs: []string{"role-1"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "role lookup error",
			userID:  "user-1",
			roleIDs: []string{"role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:    "role not found",
			userID:  "user-1",
			roleIDs: []string{"role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:    "assigner role lookup error",
			userID:  "user-1",
			roleIDs: []string{"role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return(([]types.UserRoleInfo)(nil), assignerErr).Once()
			},
			wantErr: assignerErr,
		},
		{
			name:    "replace error",
			userID:  "user-1",
			roleIDs: []string{"role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-9", RoleWeight: 20}}, nil).Once()
				userRolesRepo.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-1"}, &assignedBy).Return(replaceErr).Once()
			},
			wantErr: replaceErr,
		},
		{
			name:    "skips blank ids and succeeds with assigner",
			userID:  "user-1",
			roleIDs: []string{"", "role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-9", RoleWeight: 20}}, nil).Once()
				userRolesRepo.On("ReplaceUserRoles", mock.Anything, "user-1", []string{"role-1"}, &assignedBy).Return(nil).Once()
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
			err := service.ReplaceUserRoles(context.Background(), tc.userID, tc.roleIDs, &assignedBy)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			userRolesRepo.AssertExpectations(t)
			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestUserRolesServiceAssignRoleToUserErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")
	assignerErr := errors.New("assigner lookup failed")
	assignErr := errors.New("assign failed")
	assignedBy := "assigner-1"

	tests := []struct {
		name    string
		userID  string
		req     types.AssignUserRoleRequest
		setup   func(*accesscontroltests.MockUserRolesRepository, *accesscontroltests.MockRolesRepository)
		wantErr error
	}{
		{
			name:    "blank user id",
			userID:  "",
			req:     types.AssignUserRoleRequest{RoleID: "role-1"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "blank role id",
			userID:  "user-1",
			req:     types.AssignUserRoleRequest{},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "role lookup error",
			userID: "user-1",
			req:    types.AssignUserRoleRequest{RoleID: "role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:   "role not found",
			userID: "user-1",
			req:    types.AssignUserRoleRequest{RoleID: "role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return((*types.Role)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:   "assigner lookup error",
			userID: "user-1",
			req:    types.AssignUserRoleRequest{RoleID: "role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return(([]types.UserRoleInfo)(nil), assignerErr).Once()
			},
			wantErr: assignerErr,
		},
		{
			name:   "assign error",
			userID: "user-1",
			req:    types.AssignUserRoleRequest{RoleID: "role-1"},
			setup: func(userRolesRepo *accesscontroltests.MockUserRolesRepository, rolesRepo *accesscontroltests.MockRolesRepository) {
				rolesRepo.On("GetRoleByID", mock.Anything, "role-1").Return(&types.Role{ID: "role-1", Weight: 10}, nil).Once()
				userRolesRepo.On("GetUserRoles", mock.Anything, "assigner-1").Return([]types.UserRoleInfo{{RoleID: "role-9", RoleWeight: 20}}, nil).Once()
				userRolesRepo.On("AssignUserRole", mock.Anything, "user-1", "role-1", &assignedBy, (*time.Time)(nil)).Return(assignErr).Once()
			},
			wantErr: assignErr,
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
			err := service.AssignRoleToUser(context.Background(), tc.userID, tc.req, &assignedBy)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			userRolesRepo.AssertExpectations(t)
			rolesRepo.AssertExpectations(t)
		})
	}
}

func TestDetermineHighestActiveRoleWeight(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	expired := now.Add(-time.Hour)

	tests := []struct {
		name            string
		roles           []types.UserRoleInfo
		wantWeight      int
		wantActiveCount int
	}{
		{
			name:            "empty",
			roles:           nil,
			wantWeight:      0,
			wantActiveCount: 0,
		},
		{
			name: "ignores expired roles",
			roles: []types.UserRoleInfo{
				{RoleID: "role-1", RoleWeight: 90, ExpiresAt: &expired},
				{RoleID: "role-2", RoleWeight: 10},
				{RoleID: "role-3", RoleWeight: 30},
			},
			wantWeight:      30,
			wantActiveCount: 2,
		},
		{
			name: "all expired",
			roles: []types.UserRoleInfo{
				{RoleID: "role-1", RoleWeight: 90, ExpiresAt: &expired},
			},
			wantWeight:      0,
			wantActiveCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			weight, activeCount := determineHighestActiveRoleWeight(tc.roles, now)
			if weight != tc.wantWeight || activeCount != tc.wantActiveCount {
				t.Fatalf("expected weight=%d active=%d, got weight=%d active=%d", tc.wantWeight, tc.wantActiveCount, weight, activeCount)
			}
		})
	}
}
