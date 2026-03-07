package usecases_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	admintests "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/tests"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

func TestUsersUseCase_Create(t *testing.T) {
	t.Parallel()

	t.Run("bad request when name is empty", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		_, err := useCase.Create(context.Background(), admintypes.CreateUserRequest{Name: "", Email: "foo@bar.com"})
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("bad request when email is empty", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		_, err := useCase.Create(context.Background(), admintypes.CreateUserRequest{Name: "Name", Email: ""})
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("trims input and defaults emailVerified", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		// capture values that service sees via repo mocks
		var seenEmail string
		var seenName string
		repo.On("GetByEmail", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			seenEmail = args.String(1)
		}).Return(nil, nil).Once()

		repo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Run(func(args mock.Arguments) {
			u := args.Get(1).(*models.User)
			seenName = u.Name
			assert.False(t, u.EmailVerified, "emailVerified should default to false")
		}).Return(&models.User{ID: "user-1"}, nil).Once()

		req := admintypes.CreateUserRequest{
			Name:  "   Alice   ",
			Email: "  ALICE@EXAMPLE.COM  ",
		}

		u, err := useCase.Create(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, "Alice", seenName)
		assert.Equal(t, "alice@example.com", seenEmail)
		repo.AssertExpectations(t)
	})
}

func TestUsersUseCase_GetAll(t *testing.T) {
	t.Parallel()

	t.Run("defaults limit to 10 and trims cursor", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		var seenCursor *string
		repo.On("GetAll", mock.Anything, mock.Anything, 10).Run(func(args mock.Arguments) {
			if args.Get(1) != nil {
				str := args.Get(1).(*string)
				seenCursor = str
			}
		}).Return([]models.User{{ID: "u1"}}, (*string)(nil), nil).Once()

		cursor := "  cur-1  "
		page, err := useCase.GetAll(context.Background(), &cursor, 0)
		assert.NoError(t, err)
		assert.NotNil(t, page)
		assert.Len(t, page.Users, 1)
		if assert.NotNil(t, seenCursor) {
			assert.Equal(t, "cur-1", *seenCursor)
		}
		repo.AssertExpectations(t)
	})
}

func TestUsersUseCase_GetByID(t *testing.T) {
	t.Parallel()

	t.Run("returns error when id is empty", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		_, err := useCase.GetByID(context.Background(), "   ")
		assert.ErrorIs(t, err, constants.ErrUserIDRequired)
	})

	t.Run("forwards to service on success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		expected := &models.User{ID: "u1"}
		repo.On("GetByID", mock.Anything, "u1").Return(expected, nil).Once()

		u, err := useCase.GetByID(context.Background(), "u1")
		assert.NoError(t, err)
		assert.Equal(t, expected, u)
		repo.AssertExpectations(t)
	})
}

func TestUsersUseCase_Update(t *testing.T) {
	t.Parallel()

	t.Run("errors on empty id", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		_, err := useCase.Update(context.Background(), "", admintypes.UpdateUserRequest{})
		assert.ErrorIs(t, err, constants.ErrUserIDRequired)
	})

	t.Run("errors when nothing to update", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		_, err := useCase.Update(context.Background(), "u1", admintypes.UpdateUserRequest{})
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards changes to service", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		existing := &models.User{ID: "u1", Name: "Old", Email: "old@example.com"}
		repo.On("GetByID", mock.Anything, "u1").Return(existing, nil).Once()

		repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Run(func(args mock.Arguments) {
			u := args.Get(1).(*models.User)
			assert.Equal(t, "NewName", u.Name)
		}).Return(&models.User{ID: "u1", Name: "NewName"}, nil).Once()

		req := admintypes.UpdateUserRequest{Name: new(string)}
		*req.Name = "NewName"
		u, err := useCase.Update(context.Background(), "u1", req)
		assert.NoError(t, err)
		assert.Equal(t, "u1", u.ID)
		assert.Equal(t, "NewName", u.Name)
		repo.AssertExpectations(t)
	})
}

func TestUsersUseCase_Delete(t *testing.T) {
	t.Parallel()

	t.Run("errors when id empty", func(t *testing.T) {
		t.Parallel()

		useCase, _ := admintests.NewUsersUseCaseFixture()
		err := useCase.Delete(context.Background(), "  ")
		assert.ErrorIs(t, err, constants.ErrBadRequest)
	})

	t.Run("forwards to service on success", func(t *testing.T) {
		t.Parallel()

		useCase, repo := admintests.NewUsersUseCaseFixture()
		repo.On("GetByID", mock.Anything, "u1").Return(&models.User{ID: "u1"}, nil).Once()
		repo.On("Delete", mock.Anything, "u1").Return(nil).Once()

		err := useCase.Delete(context.Background(), "u1")
		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}
