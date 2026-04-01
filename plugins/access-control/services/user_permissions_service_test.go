package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	accesscontroltests "github.com/Authula/authula/plugins/access-control/tests"
	"github.com/Authula/authula/plugins/access-control/types"
)

func TestUserPermissionsServiceGetUserPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        string
		setupMock     func(*accesscontroltests.MockUserPermissionsRepository)
		expectedCount int
		expectedErr   error
	}{
		{
			name:        "blank user id",
			userID:      "",
			expectedErr: accesscontrolconstants.ErrUnprocessableEntity,
		},
		{
			name:   "success",
			userID: "u1",
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("GetUserPermissions", mock.Anything, "u1").Return([]types.UserPermissionInfo{{PermissionID: "perm-1", PermissionKey: "users.read"}}, nil).Once()
			},
			expectedCount: 1,
		},
		{
			name:   "repo error",
			userID: "u1",
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("GetUserPermissions", mock.Anything, "u1").Return(([]types.UserPermissionInfo)(nil), accesscontrolconstants.ErrNotFound).Once()
			},
			expectedErr: accesscontrolconstants.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &accesscontroltests.MockUserPermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}

			service := NewUserPermissionsService(repo)
			permissions, err := service.GetUserPermissions(context.Background(), tc.userID)
			if tc.expectedErr != nil {
				if err != tc.expectedErr {
					t.Fatalf("expected err %v, got %v", tc.expectedErr, err)
				}
				repo.AssertExpectations(t)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(permissions) != tc.expectedCount {
				t.Fatalf("expected %d permissions, got %d", tc.expectedCount, len(permissions))
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserPermissionsServiceHasPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		userID         string
		permissionKeys []string
		setupMock      func(*accesscontroltests.MockUserPermissionsRepository)
		expectedHas    bool
		expectedErr    error
	}{
		{
			name:        "blank user id",
			userID:      "",
			expectedErr: accesscontrolconstants.ErrUnprocessableEntity,
		},
		{
			name:           "success",
			userID:         "u1",
			permissionKeys: []string{"users.read"},
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("HasPermissions", mock.Anything, "u1", []string{"users.read"}).Return(true, nil).Once()
			},
			expectedHas: true,
		},
		{
			name:           "repo error",
			userID:         "u1",
			permissionKeys: []string{"users.read"},
			setupMock: func(m *accesscontroltests.MockUserPermissionsRepository) {
				m.On("HasPermissions", mock.Anything, "u1", []string{"users.read"}).Return(false, accesscontrolconstants.ErrForbidden).Once()
			},
			expectedErr: accesscontrolconstants.ErrForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &accesscontroltests.MockUserPermissionsRepository{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}

			service := NewUserPermissionsService(repo)
			hasPermissions, err := service.HasPermissions(context.Background(), tc.userID, tc.permissionKeys)
			if tc.expectedErr != nil {
				if err != tc.expectedErr {
					t.Fatalf("expected err %v, got %v", tc.expectedErr, err)
				}
				repo.AssertExpectations(t)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasPermissions != tc.expectedHas {
				t.Fatalf("expected hasPermissions=%v, got %v", tc.expectedHas, hasPermissions)
			}
			repo.AssertExpectations(t)
		})
	}
}
