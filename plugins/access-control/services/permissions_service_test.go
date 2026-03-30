package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestPermissionsServiceCreatePermission(t *testing.T) {
	t.Parallel()

	description := "Read users"

	tests := []struct {
		name    string
		req     types.CreatePermissionRequest
		setup   func(*accesscontroltests.MockPermissionsRepository)
		wantErr error
		assert  func(*testing.T, *types.Permission)
	}{
		{
			name:    "blank key",
			req:     types.CreatePermissionRequest{Key: ""},
			wantErr: accesscontrolconstants.ErrBadRequest,
		},
		{
			name: "success",
			req: types.CreatePermissionRequest{
				Key:         "users.read",
				Description: &description,
				IsSystem:    true,
			},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("CreatePermission", mock.Anything, mock.MatchedBy(func(permission *types.Permission) bool {
					return permission != nil && permission.ID != "" && permission.Key == "users.read" && permission.IsSystem && permission.Description != nil && *permission.Description == description
				})).Return(nil).Once()
			},
			assert: func(t *testing.T, permission *types.Permission) {
				if permission == nil {
					t.Fatal("expected permission, got nil")
				}
				if permission.ID == "" {
					t.Fatal("expected generated ID")
				}
				if permission.Key != "users.read" {
					t.Fatalf("expected key %q, got %q", "users.read", permission.Key)
				}
				if permission.Description == nil || *permission.Description != description {
					t.Fatalf("expected description %q, got %#v", description, permission.Description)
				}
				if !permission.IsSystem {
					t.Fatal("expected system permission flag to be preserved")
				}
			},
		},
		{
			name: "repository error is returned",
			req:  types.CreatePermissionRequest{Key: "users.write"},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("CreatePermission", mock.Anything, mock.AnythingOfType("*types.Permission")).Return(errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, userAccessRepo)
			permission, err := service.CreatePermission(context.Background(), tc.req)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil err, got %v", err)
				}
			} else if err == nil || err.Error() != tc.wantErr.Error() {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			if tc.assert != nil {
				tc.assert(t, permission)
			}

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionsServiceGetAllPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockPermissionsRepository)
		want    []types.Permission
		wantErr error
	}{
		{
			name: "success",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetAllPermissions", mock.Anything).Return([]types.Permission{{ID: "perm-1", Key: "users.read"}}, nil).Once()
			},
			want: []types.Permission{{ID: "perm-1", Key: "users.read"}},
		},
		{
			name: "error",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetAllPermissions", mock.Anything).Return(nil, errors.New("boom")).Once()
			},
			wantErr: errors.New("boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, userAccessRepo)
			permissions, err := service.GetAllPermissions(context.Background())
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil err, got %v", err)
				}
				if len(permissions) != len(tc.want) || (len(permissions) == 1 && permissions[0].ID != tc.want[0].ID) {
					t.Fatalf("unexpected permissions %#v", permissions)
				}
			} else if err == nil || err.Error() != tc.wantErr.Error() {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionsServiceGetPermissionByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		setup   func(*accesscontroltests.MockPermissionsRepository)
		wantID  string
		wantErr error
	}{
		{
			name:    "blank id",
			id:      "",
			wantErr: accesscontrolconstants.ErrBadRequest,
		},
		{
			name: "not found",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: accesscontrolconstants.ErrNotFound,
		},
		{
			name: "success",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
			},
			wantID: "perm-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, userAccessRepo)
			permission, err := service.GetPermissionByID(context.Background(), tc.id)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil err, got %v", err)
				}
				if permission == nil || permission.ID != tc.wantID {
					t.Fatalf("unexpected permission %#v", permission)
				}
			} else if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionsServiceUpdatePermission(t *testing.T) {
	t.Parallel()

	updatedDescription := "Updated description"

	tests := []struct {
		name    string
		id      string
		req     types.UpdatePermissionRequest
		setup   func(*accesscontroltests.MockPermissionsRepository)
		wantID  string
		wantErr error
	}{
		{
			name:    "blank id",
			id:      "",
			req:     types.UpdatePermissionRequest{Description: &updatedDescription},
			wantErr: accesscontrolconstants.ErrUnprocessableEntity,
		},
		{
			name:    "nil description",
			id:      "perm-1",
			req:     types.UpdatePermissionRequest{},
			wantErr: accesscontrolconstants.ErrUnprocessableEntity,
		},
		{
			name:    "blank description",
			id:      "perm-1",
			req:     types.UpdatePermissionRequest{Description: func() *string { value := ""; return &value }()},
			wantErr: accesscontrolconstants.ErrUnprocessableEntity,
		},
		{
			name: "not found",
			id:   "perm-1",
			req:  types.UpdatePermissionRequest{Description: &updatedDescription},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: accesscontrolconstants.ErrNotFound,
		},
		{
			name: "system permission",
			id:   "perm-1",
			req:  types.UpdatePermissionRequest{Description: &updatedDescription},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", IsSystem: true}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrBadRequest,
		},
		{
			name: "update returns false",
			id:   "perm-1",
			req:  types.UpdatePermissionRequest{Description: &updatedDescription},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				permissionsRepo.On("UpdatePermission", mock.Anything, "perm-1", &updatedDescription).Return(false, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrNotFound,
		},
		{
			name: "success",
			id:   "perm-1",
			req:  types.UpdatePermissionRequest{Description: &updatedDescription},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				permissionsRepo.On("UpdatePermission", mock.Anything, "perm-1", &updatedDescription).Return(true, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", Description: &updatedDescription}, nil).Once()
			},
			wantID: "perm-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, userAccessRepo)
			permission, err := service.UpdatePermission(context.Background(), tc.id, tc.req)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil err, got %v", err)
				}
				if permission == nil || permission.ID != tc.wantID {
					t.Fatalf("unexpected permission %#v", permission)
				}
			} else if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionsServiceDeletePermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		setup   func(*accesscontroltests.MockPermissionsRepository, *accesscontroltests.MockUserAccessRepository)
		wantErr error
	}{
		{
			name:    "blank id",
			id:      "",
			wantErr: accesscontrolconstants.ErrBadRequest,
		},
		{
			name: "not found",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: accesscontrolconstants.ErrNotFound,
		},
		{
			name: "system permission",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", IsSystem: true}, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrBadRequest,
		},
		{
			name: "permission in use",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				userAccessRepo.On("CountRoleAssignmentsByPermissionID", mock.Anything, "perm-1").Return(2, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrConflict,
		},
		{
			name: "delete returns false",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				userAccessRepo.On("CountRoleAssignmentsByPermissionID", mock.Anything, "perm-1").Return(0, nil).Once()
				permissionsRepo.On("DeletePermission", mock.Anything, "perm-1").Return(false, nil).Once()
			},
			wantErr: accesscontrolconstants.ErrNotFound,
		},
		{
			name: "success",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, userAccessRepo *accesscontroltests.MockUserAccessRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				userAccessRepo.On("CountRoleAssignmentsByPermissionID", mock.Anything, "perm-1").Return(0, nil).Once()
				permissionsRepo.On("DeletePermission", mock.Anything, "perm-1").Return(true, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			userAccessRepo := &accesscontroltests.MockUserAccessRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo, userAccessRepo)
			}

			service := NewPermissionsService(permissionsRepo, userAccessRepo)
			err := service.DeletePermission(context.Background(), tc.id)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil err, got %v", err)
				}
			} else if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
			userAccessRepo.AssertExpectations(t)
		})
	}
}
