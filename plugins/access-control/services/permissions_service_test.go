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
			wantErr: coreerrors.ErrBadRequest,
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
				} else {
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
				}
			},
		},
		{
			name: "repository error is returned",
			req:  types.CreatePermissionRequest{Key: "users.write"},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("CreatePermission", mock.Anything, mock.AnythingOfType("*types.Permission")).Return(errors.New("some error")).Once()
			},
			wantErr: errors.New("some error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			_ = rolePermissionsRepo
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, rolePermissionsRepo)
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
				permissionsRepo.On("GetAllPermissions", mock.Anything).Return(nil, errors.New("some error")).Once()
			},
			wantErr: errors.New("some error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			_ = rolePermissionsRepo
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, rolePermissionsRepo)
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
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "not found",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
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
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			_ = rolePermissionsRepo
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, rolePermissionsRepo)
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
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:    "nil description",
			id:      "perm-1",
			req:     types.UpdatePermissionRequest{},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name:    "blank description",
			id:      "perm-1",
			req:     types.UpdatePermissionRequest{Description: func() *string { value := ""; return &value }()},
			wantErr: coreerrors.ErrUnprocessableEntity,
		},
		{
			name: "not found",
			id:   "perm-1",
			req:  types.UpdatePermissionRequest{Description: &updatedDescription},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name: "system permission",
			id:   "perm-1",
			req:  types.UpdatePermissionRequest{Description: &updatedDescription},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", IsSystem: true}, nil).Once()
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "update returns false",
			id:   "perm-1",
			req:  types.UpdatePermissionRequest{Description: &updatedDescription},
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				permissionsRepo.On("UpdatePermission", mock.Anything, "perm-1", &updatedDescription).Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
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
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, rolePermissionsRepo)
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
		})
	}
}

func TestPermissionsServiceDeletePermission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		setup   func(*accesscontroltests.MockPermissionsRepository, *accesscontroltests.MockRolePermissionsRepository)
		wantErr error
	}{
		{
			name:    "blank id",
			id:      "",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "not found",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name: "system permission",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read", IsSystem: true}, nil).Once()
			},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "permission in use",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				rolePermissionsRepo.On("CountRolesByPermission", mock.Anything, "perm-1").Return(2, nil).Once()
			},
			wantErr: coreerrors.ErrConflict,
		},
		{
			name: "delete returns false",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				rolePermissionsRepo.On("CountRolesByPermission", mock.Anything, "perm-1").Return(0, nil).Once()
				permissionsRepo.On("DeletePermission", mock.Anything, "perm-1").Return(false, nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name: "success",
			id:   "perm-1",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
				rolePermissionsRepo.On("CountRolesByPermission", mock.Anything, "perm-1").Return(0, nil).Once()
				permissionsRepo.On("DeletePermission", mock.Anything, "perm-1").Return(true, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo, rolePermissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, rolePermissionsRepo)
			err := service.DeletePermission(context.Background(), tc.id)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil err, got %v", err)
				}
			} else if err != tc.wantErr {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionsServiceGetPermissionByKey(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")

	tests := []struct {
		name    string
		key     string
		setup   func(*accesscontroltests.MockPermissionsRepository)
		wantID  string
		wantErr error
	}{
		{
			name:    "blank key",
			key:     "",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name: "repository error",
			key:  "users.read",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "users.read").Return((*types.Permission)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name: "not found",
			key:  "users.read",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "users.read").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name: "success",
			key:  "users.read",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByKey", mock.Anything, "users.read").Return(&types.Permission{ID: "perm-1", Key: "users.read"}, nil).Once()
			},
			wantID: "perm-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			if tc.setup != nil {
				tc.setup(permissionsRepo)
			}

			service := NewPermissionsService(permissionsRepo, nil)
			permission, err := service.GetPermissionByKey(context.Background(), tc.key)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if permission != nil {
					t.Fatalf("expected nil permission, got %#v", permission)
				}
				permissionsRepo.AssertExpectations(t)
				return
			}
			if permission == nil || permission.ID != tc.wantID {
				t.Fatalf("unexpected permission %#v", permission)
			}

			permissionsRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionsServiceGetPermissionByIDRepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")

	permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
	permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), repoErr).Once()

	service := NewPermissionsService(permissionsRepo, nil)
	permission, err := service.GetPermissionByID(context.Background(), "perm-1")
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected err %v, got %v", repoErr, err)
	}
	if permission != nil {
		t.Fatalf("expected nil permission, got %#v", permission)
	}

	permissionsRepo.AssertExpectations(t)
}

func TestPermissionsServiceUpdatePermissionRepositoryErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")
	updateErr := errors.New("update failed")
	refetchErr := errors.New("refetch failed")
	description := "Updated description"

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockPermissionsRepository)
		wantErr error
	}{
		{
			name: "lookup error",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name: "update error",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				permissionsRepo.On("UpdatePermission", mock.Anything, "perm-1", &description).Return(false, updateErr).Once()
			},
			wantErr: updateErr,
		},
		{
			name: "refetch error",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				permissionsRepo.On("UpdatePermission", mock.Anything, "perm-1", &description).Return(true, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), refetchErr).Once()
			},
			wantErr: refetchErr,
		},
		{
			name: "refetch returns nil",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				permissionsRepo.On("UpdatePermission", mock.Anything, "perm-1", &description).Return(true, nil).Once()
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			tc.setup(permissionsRepo)

			service := NewPermissionsService(permissionsRepo, nil)
			permission, err := service.UpdatePermission(context.Background(), "perm-1", types.UpdatePermissionRequest{Description: &description})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}
			if permission != nil {
				t.Fatalf("expected nil permission, got %#v", permission)
			}

			permissionsRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionsServiceDeletePermissionRepositoryErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")
	countErr := errors.New("count failed")
	deleteErr := errors.New("delete failed")

	tests := []struct {
		name    string
		setup   func(*accesscontroltests.MockPermissionsRepository, *accesscontroltests.MockRolePermissionsRepository)
		wantErr error
	}{
		{
			name: "lookup error",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return((*types.Permission)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name: "count error",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				rolePermissionsRepo.On("CountRolesByPermission", mock.Anything, "perm-1").Return(0, countErr).Once()
			},
			wantErr: countErr,
		},
		{
			name: "delete error",
			setup: func(permissionsRepo *accesscontroltests.MockPermissionsRepository, rolePermissionsRepo *accesscontroltests.MockRolePermissionsRepository) {
				permissionsRepo.On("GetPermissionByID", mock.Anything, "perm-1").Return(&types.Permission{ID: "perm-1"}, nil).Once()
				rolePermissionsRepo.On("CountRolesByPermission", mock.Anything, "perm-1").Return(0, nil).Once()
				permissionsRepo.On("DeletePermission", mock.Anything, "perm-1").Return(false, deleteErr).Once()
			},
			wantErr: deleteErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			permissionsRepo := &accesscontroltests.MockPermissionsRepository{}
			rolePermissionsRepo := &accesscontroltests.MockRolePermissionsRepository{}
			tc.setup(permissionsRepo, rolePermissionsRepo)

			service := NewPermissionsService(permissionsRepo, rolePermissionsRepo)
			err := service.DeletePermission(context.Background(), "perm-1")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected err %v, got %v", tc.wantErr, err)
			}

			permissionsRepo.AssertExpectations(t)
			rolePermissionsRepo.AssertExpectations(t)
		})
	}
}
