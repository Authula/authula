package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func newUsersServiceFixture() (*adminservices.UsersService, *internaltests.MockUserRepository) {
	repo := &internaltests.MockUserRepository{}
	return adminservices.NewUsersService(repo), repo
}

func TestUsersService_Create(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("error")
	createErr := errors.New("fail")

	tests := []struct {
		name    string
		request admintypes.CreateUserRequest
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:    "missing name",
			request: admintypes.CreateUserRequest{Email: "a@b"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "missing email",
			request: admintypes.CreateUserRequest{Name: "n"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "email conflict",
			request: admintypes.CreateUserRequest{Email: "a@b", Name: "n"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "a@b").Return(&models.User{Email: "a@b"}, nil).Once()
			},
			wantErr: coreerrors.ErrConflict,
		},
		{
			name:    "repo get error",
			request: admintypes.CreateUserRequest{Email: "a@b", Name: "n"},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "a@b").Return((*models.User)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:    "create failure",
			request: admintypes.CreateUserRequest{Email: "a@b", Name: "n", EmailVerified: new(true)},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "a@b").Return((*models.User)(nil), nil).Once()
				repo.On("Create", mock.Anything, mock.Anything).Return(&models.User{Email: "a@b"}, createErr).Once()
			},
			wantErr: createErr,
		},
		{
			name:    "success",
			request: admintypes.CreateUserRequest{Email: "a@b", Name: "n", EmailVerified: new(true)},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "a@b").Return((*models.User)(nil), nil).Once()
				repo.On("Create", mock.Anything, mock.Anything).Return(&models.User{Email: "a@b"}, nil).Once()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			if tc.setup != nil {
				tc.setup(repo)
			}

			user, err := svc.Create(context.Background(), internaltests.TestActor(), tc.request)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tc.request.Email, user.Email)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestUsersService_GetAll(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")

	tests := []struct {
		name    string
		cursor  *string
		limit   int
		setup   func(repo *internaltests.MockUserRepository)
		wantLen int
		wantErr error
	}{
		{
			name:  "success",
			limit: 10,
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetAll", mock.Anything, mock.Anything, 10).Return([]models.User{{Email: "a"}}, (*string)(nil), nil).Once()
			},
			wantLen: 1,
		},
		{
			name:   "defaults limit to 10 and trims cursor",
			cursor: new("  cur-1  "),
			limit:  0,
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetAll", mock.Anything, mock.Anything, 10).Return([]models.User{{Email: "a"}}, (*string)(nil), nil).Once()
			},
			wantLen: 1,
		},
		{
			name:  "repo error",
			limit: 5,
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetAll", mock.Anything, mock.Anything, 5).Return(([]models.User)(nil), (*string)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			if tc.setup != nil {
				tc.setup(repo)
			}

			page, err := svc.GetAll(context.Background(), internaltests.TestActor(), tc.cursor, tc.limit)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, page)
			} else {
				assert.NoError(t, err)
				assert.Len(t, page.Users, tc.wantLen)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUsersService_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
			},
		},
		{
			name:    "missing user id",
			userID:  "   ",
			wantErr: adminconstants.ErrUserIDRequired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			if tc.setup != nil {
				tc.setup(repo)
			}

			u, err := svc.GetByID(context.Background(), internaltests.TestActor(), tc.userID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, u)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.userID, u.ID)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUsersService_Update(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	updateErr := errors.New("update failure")

	tests := []struct {
		name       string
		userID     string
		request    admintypes.UpdateUserRequest
		setup      func(repo *internaltests.MockUserRepository)
		wantErr    error
		assertUser func(t *testing.T, user *models.User)
	}{
		{
			name:    "success",
			userID:  "u1",
			request: admintypes.UpdateUserRequest{Email: new("x"), Name: new("y"), EmailVerified: new(true)},
			setup: func(repo *internaltests.MockUserRepository) {
				base := &models.User{ID: "u1", Email: "e", Name: "n", EmailVerified: false}
				repo.On("GetByID", mock.Anything, "u1").Return(base, nil).Once()
				repo.On("Update", mock.Anything, mock.Anything).Return(base, nil).Once()
			},
			assertUser: func(t *testing.T, user *models.User) {
				t.Helper()
				assert.Equal(t, "x", user.Email)
				assert.Equal(t, "y", user.Name)
				assert.True(t, user.EmailVerified)
			},
		},
		{
			name:    "not found",
			userID:  "u1",
			request: admintypes.UpdateUserRequest{Email: new("x")},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:    "missing user id",
			userID:  "   ",
			request: admintypes.UpdateUserRequest{Email: new("x")},
			wantErr: adminconstants.ErrUserIDRequired,
		},
		{
			name:    "nothing to update",
			userID:  "u1",
			request: admintypes.UpdateUserRequest{},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "lookup fails",
			userID:  " u1 ",
			request: admintypes.UpdateUserRequest{Name: new("name")},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:    "update fails",
			userID:  " u1 ",
			request: admintypes.UpdateUserRequest{Name: new("name")},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return((*models.User)(nil), updateErr).Once()
			},
			wantErr: updateErr,
		},
		{
			name:   "applies image and metadata",
			userID: " u1 ",
			request: admintypes.UpdateUserRequest{
				Image:    new("avatar.png"),
				Metadata: map[string]any{"role": "admin"},
			},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				repo.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
					return u.Image != nil && *u.Image == "avatar.png" &&
						u.Metadata["role"] == "admin"
				})).Return(&models.User{ID: "u1", Image: new("avatar.png")}, nil).Once()
			},
			assertUser: func(t *testing.T, user *models.User) {
				t.Helper()
				require.NotNil(t, user.Image)
				assert.Equal(t, "avatar.png", *user.Image)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			if tc.setup != nil {
				tc.setup(repo)
			}

			updated, err := svc.Update(context.Background(), internaltests.TestActor(), tc.userID, tc.request)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, updated)
			} else {
				require.NoError(t, err)
				require.NotNil(t, updated)
				if tc.assertUser != nil {
					tc.assertUser(t, updated)
				}
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUsersService_Delete(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("lookup failure")
	deleteErr := errors.New("delete failure")

	tests := []struct {
		name    string
		userID  string
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:   "success",
			userID: "u1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				repo.On("Delete", mock.Anything, "u1").Return(nil).Once()
			},
		},
		{
			name:   "not found",
			userID: "u1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), nil).Once()
			},
			wantErr: coreerrors.ErrNotFound,
		},
		{
			name:    "missing user id",
			userID:  "   ",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "lookup fails",
			userID: "u1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), lookupErr).Once()
			},
			wantErr: lookupErr,
		},
		{
			name:   "delete fails",
			userID: "u1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				repo.On("Delete", mock.Anything, "u1").Return(deleteErr).Once()
			},
			wantErr: deleteErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			if tc.setup != nil {
				tc.setup(repo)
			}

			err := svc.Delete(context.Background(), internaltests.TestActor(), tc.userID)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
