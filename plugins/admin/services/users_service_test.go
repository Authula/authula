package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminconstants "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	adminservices "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func newUsersServiceFixture() (*adminservices.UsersService, *internaltests.MockUserRepository) {
	repo := &internaltests.MockUserRepository{}
	return adminservices.NewUsersService(repo), repo
}

func TestUsersService_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name         string
		existing     *models.User
		repoErr      error
		createErr    error
		request      admintypes.CreateUserRequest
		wantErr      error
		expectCreate bool
	}{
		{
			name:     "email conflict",
			existing: &models.User{Email: "a@b"},
			request:  admintypes.CreateUserRequest{Email: "a@b"},
			wantErr:  adminconstants.ErrConflict,
		},
		{
			name:    "repo get error",
			repoErr: errors.New("error"),
			request: admintypes.CreateUserRequest{Email: "a@b"},
		},
		{
			name:         "create failure",
			existing:     nil,
			createErr:    errors.New("fail"),
			request:      admintypes.CreateUserRequest{Email: "a@b", Name: "n", EmailVerified: func(b bool) *bool { return &b }(true)},
			expectCreate: true,
		},
		{
			name:         "success",
			existing:     nil,
			request:      admintypes.CreateUserRequest{Email: "a@b", Name: "n", EmailVerified: func(b bool) *bool { return &b }(true)},
			expectCreate: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			repo.ExpectedCalls = nil
			repo.On("GetByEmail", mock.Anything, tc.request.Email).Return(tc.existing, tc.repoErr).Once()
			if tc.expectCreate {
				// return a simple user with matching email
				repo.On("Create", mock.Anything, mock.Anything).
					Return(&models.User{Email: tc.request.Email}, tc.createErr).
					Once()
			}

			user, err := svc.Create(ctx, tc.request)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, user)
			} else if tc.repoErr != nil || tc.createErr != nil {
				assert.Error(t, err)
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

	svc, repo := newUsersServiceFixture()
	ctx := context.Background()

	repo.On("GetAll", mock.Anything, (*string)(nil), 10).Return([]models.User{{Email: "a"}}, nil, nil).Once()

	page, err := svc.GetAll(ctx, nil, 10)
	assert.NoError(t, err)
	assert.Len(t, page.Users, 1)
	repo.AssertExpectations(t)
}

func TestUsersService_GetByID(t *testing.T) {
	t.Parallel()

	svc, repo := newUsersServiceFixture()
	ctx := context.Background()

	repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	u, err := svc.GetByID(ctx, "u1")
	assert.NoError(t, err)
	assert.Equal(t, "u1", u.ID)
	repo.AssertExpectations(t)
}

func TestUsersService_Update(t *testing.T) {
	t.Parallel()

	svc, repo := newUsersServiceFixture()
	ctx := context.Background()

	base := &models.User{ID: "u1", Email: "e", Name: "n", EmailVerified: false}
	repo.On("GetByID", mock.Anything, "u1").Return(base, nil).Once()
	// return same base so modifications are visible in result
	repo.On("Update", mock.Anything, mock.Anything).Return(base, nil).Once()

	req := admintypes.UpdateUserRequest{Email: new("x"), Name: new("y"), EmailVerified: func(b bool) *bool { return &b }(true)}
	updated, err := svc.Update(ctx, "u1", req)
	assert.NoError(t, err)
	assert.Equal(t, "x", updated.Email)
	assert.Equal(t, "y", updated.Name)
	assert.True(t, updated.EmailVerified)
	repo.AssertExpectations(t)
}

func TestUsersService_Update_notFound(t *testing.T) {
	t.Parallel()

	svc, repo := newUsersServiceFixture()
	ctx := context.Background()
	repo.On("GetByID", mock.Anything, "u1").Return(nil, nil).Once()

	_, err := svc.Update(ctx, "u1", admintypes.UpdateUserRequest{})
	assert.ErrorIs(t, err, adminconstants.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestUsersService_Delete(t *testing.T) {
	t.Parallel()

	svc, repo := newUsersServiceFixture()
	ctx := context.Background()

	repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
	repo.On("Delete", mock.Anything, "u1").Return(nil).Once()
	assert.NoError(t, svc.Delete(ctx, "u1"))
	repo.AssertExpectations(t)
}

func TestUsersService_Delete_notFound(t *testing.T) {
	t.Parallel()

	svc, repo := newUsersServiceFixture()
	ctx := context.Background()
	repo.On("GetByID", mock.Anything, "u1").Return(nil, nil).Once()
	assert.ErrorIs(t, svc.Delete(ctx, "u1"), adminconstants.ErrNotFound)
	repo.AssertExpectations(t)
}
