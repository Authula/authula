package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	admintypes "github.com/Authula/authula/plugins/admin/types"
)

func TestUsersUseCase_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request admintypes.CreateUserRequest
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:    "bad request when name is empty",
			request: admintypes.CreateUserRequest{Name: "", Email: "foo@bar.com"},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "bad request when email is empty",
			request: admintypes.CreateUserRequest{Name: "Name", Email: ""},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "trims input and defaults emailVerified",
			request: admintypes.CreateUserRequest{Name: "   Alice   ", Email: "  ALICE@EXAMPLE.COM  "},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "alice@example.com").Return((*models.User)(nil), nil).Once()
				repo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Run(func(args mock.Arguments) {
					u := args.Get(1).(*models.User)
					assert.Equal(t, "Alice", u.Name)
					assert.Equal(t, "alice@example.com", u.Email)
					assert.False(t, u.EmailVerified, "emailVerified should default to false")
				}).Return(&models.User{ID: "user-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			if tt.setup != nil {
				tt.setup(repo)
			}

			u, err := useCase.Create(context.Background(), internaltests.TestActor(), tt.request)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, u)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, u)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUsersUseCase_GetAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cursor  *string
		limit   int
		wantErr error
	}{
		{
			name:   "defaults limit to 10 and trims cursor",
			cursor: new("  cur-1  "),
			limit:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			repo.On("GetAll", mock.Anything, mock.MatchedBy(func(c *string) bool {
				return c != nil && *c == "cur-1"
			}), 10).Return([]models.User{{ID: "u1"}}, (*string)(nil), nil).Once()

			page, err := useCase.GetAll(context.Background(), internaltests.TestActor(), tt.cursor, tt.limit)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, page)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, page)
				assert.Len(t, page.Users, 1)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUsersUseCase_GetByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:    "returns error when id is empty",
			userID:  "   ",
			wantErr: adminconstants.ErrUserIDRequired,
		},
		{
			name:   "forwards to service on success",
			userID: "u1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			if tt.setup != nil {
				tt.setup(repo)
			}

			u, err := useCase.GetByID(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, u)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "u1", u.ID)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUsersUseCase_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		request admintypes.UpdateUserRequest
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:    "errors on empty id",
			userID:  "",
			request: admintypes.UpdateUserRequest{},
			wantErr: adminconstants.ErrUserIDRequired,
		},
		{
			name:    "errors when nothing to update",
			userID:  "u1",
			request: admintypes.UpdateUserRequest{},
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:    "forwards changes to service",
			userID:  "u1",
			request: admintypes.UpdateUserRequest{Name: new("NewName")},
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1", Name: "Old", Email: "old@example.com"}, nil).Once()
				repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Run(func(args mock.Arguments) {
					u := args.Get(1).(*models.User)
					assert.Equal(t, "NewName", u.Name)
				}).Return(&models.User{ID: "u1", Name: "NewName"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			if tt.setup != nil {
				tt.setup(repo)
			}

			u, err := useCase.Update(context.Background(), internaltests.TestActor(), tt.userID, tt.request)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, u)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "u1", u.ID)
				assert.Equal(t, "NewName", u.Name)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUsersUseCase_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		setup   func(repo *internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:    "errors when id empty",
			userID:  "  ",
			wantErr: coreerrors.ErrBadRequest,
		},
		{
			name:   "forwards to service on success",
			userID: "u1",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
				repo.On("Delete", mock.Anything, "u1").Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo := admintests.NewUsersUseCaseFixture()
			if tt.setup != nil {
				tt.setup(repo)
			}

			err := useCase.Delete(context.Background(), internaltests.TestActor(), tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
