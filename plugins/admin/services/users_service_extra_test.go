package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func TestUsersService_GetAll_RepoError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	svc, repo := newUsersServiceFixture()
	repo.On("GetAll", mock.Anything, mock.Anything, 5).Return(([]models.User)(nil), (*string)(nil), repoErr).Once()

	page, err := svc.GetAll(context.Background(), internaltests.TestActor(), nil, 5)
	assert.ErrorIs(t, err, repoErr)
	assert.Nil(t, page)

	repo.AssertExpectations(t)
}

func TestUsersService_Update_ErrorsAndFields(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repo failure")
	updateErr := errors.New("update failure")

	tests := []struct {
		name    string
		request admintypes.UpdateUserRequest
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
		assert  func(t *testing.T, user *models.User)
	}{
		{
			name:    "lookup fails",
			request: admintypes.UpdateUserRequest{Name: new("name")},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), repoErr).Once()
			},
			wantErr: repoErr,
		},
		{
			name:    "update fails",
			request: admintypes.UpdateUserRequest{Name: new("name")},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return((*models.User)(nil), updateErr).Once()
			},
			wantErr: updateErr,
		},
		{
			name: "applies image and metadata",
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
			assert: func(t *testing.T, user *models.User) {
				t.Helper()
				require.NotNil(t, user.Image)
				assert.Equal(t, "avatar.png", *user.Image)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			tt.setup(repo)

			user, err := svc.Update(context.Background(), internaltests.TestActor(), " u1 ", tt.request)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				if tt.assert != nil {
					tt.assert(t, user)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestUsersService_Delete_Errors(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("lookup failure")
	deleteErr := errors.New("delete failure")

	tests := []struct {
		name    string
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name: "lookup fails",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return((*models.User)(nil), lookupErr).Once()
			},
			wantErr: lookupErr,
		},
		{
			name: "delete fails",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				repo.On("Delete", mock.Anything, "u1").Return(deleteErr).Once()
			},
			wantErr: deleteErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newUsersServiceFixture()
			tt.setup(repo)

			err := svc.Delete(context.Background(), internaltests.TestActor(), "u1")
			assert.ErrorIs(t, err, tt.wantErr)

			repo.AssertExpectations(t)
		})
	}
}
