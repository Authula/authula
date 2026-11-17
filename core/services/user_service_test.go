package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreservices "github.com/Authula/authula/core/services"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	rootservices "github.com/Authula/authula/services"
)

func TestUserService_NormalizesEmailArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(*internaltests.MockUserRepository)
		call      func(context.Context, rootservices.UserService) (string, error)
		wantEmail string
	}{
		{
			name: "GetByEmail",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
			},
			call: func(ctx context.Context, svc rootservices.UserService) (string, error) {
				user, err := svc.GetByEmail(ctx, "  USER@Example.COM  ")
				if user == nil {
					return "", err
				}
				return user.Email, err
			},
			wantEmail: "user@example.com",
		},
		{
			name: "Create",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "user@example.com").Return(nil, nil)
				repo.On("Create", mock.Anything, mock.MatchedBy(func(user *models.User) bool {
					return user.Email == "user@example.com" && user.Name == "User"
				})).Return(&models.User{ID: "user-1", Email: "user@example.com", Name: "User"}, nil)
			},
			call: func(ctx context.Context, svc rootservices.UserService) (string, error) {
				user, err := svc.Create(ctx, "  User  ", "  USER@Example.COM  ", false, nil, nil)
				if user == nil {
					return "", err
				}
				return user.Email, err
			},
			wantEmail: "user@example.com",
		},
		{
			name: "Update",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("Update", mock.Anything, mock.MatchedBy(func(user *models.User) bool {
					return user.Email == "user@example.com"
				})).Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)
			},
			call: func(ctx context.Context, svc rootservices.UserService) (string, error) {
				user, err := svc.Update(ctx, &models.User{ID: "user-1", Email: "  USER@Example.COM  "})
				if user == nil {
					return "", err
				}
				return user.Email, err
			},
			wantEmail: "user@example.com",
		},
		{
			name: "UpdateFields",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("UpdateFields", mock.Anything, "user-1", mock.MatchedBy(func(fields map[string]any) bool {
					return fields["email"] == "user@example.com"
				})).Return(nil)
			},
			call: func(ctx context.Context, svc rootservices.UserService) (string, error) {
				return "", svc.UpdateFields(ctx, "user-1", map[string]any{"email": "  USER@Example.COM  "})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &internaltests.MockUserRepository{}
			tt.setup(repo)
			svc := coreservices.NewUserService(repo, nil, &internaltests.MockLogger{})

			got, err := tt.call(context.Background(), svc)

			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, got)
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetAll(t *testing.T) {
	t.Parallel()

	users := []models.User{{ID: "user-1"}, {ID: "user-2"}}
	nextCursor := new("cursor-2")
	repoErr := errors.New("database unavailable")

	tests := []struct {
		name      string
		cursor    *string
		limit     int
		wantLimit int
		repoErr   error
	}{
		{
			name:      "zero limit defaults to ten",
			limit:     0,
			wantLimit: 10,
		},
		{
			name:      "negative limit defaults to ten",
			limit:     -5,
			wantLimit: 10,
		},
		{
			name:      "positive limit is forwarded unchanged",
			limit:     25,
			wantLimit: 25,
		},
		{
			name:      "cursor is forwarded with the limit",
			cursor:    new("cursor-1"),
			limit:     5,
			wantLimit: 5,
		},
		{
			name:      "repository error is returned",
			limit:     10,
			wantLimit: 10,
			repoErr:   repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockUserRepository)
			if tt.repoErr != nil {
				repo.On("GetAll", mock.Anything, tt.cursor, tt.wantLimit).Return(nil, nil, tt.repoErr).Once()
			} else {
				repo.On("GetAll", mock.Anything, tt.cursor, tt.wantLimit).Return(users, nextCursor, nil).Once()
			}
			svc := coreservices.NewUserService(repo, nil, new(internaltests.MockLogger))

			got, gotCursor, err := svc.GetAll(context.Background(), tt.cursor, tt.limit)

			if tt.repoErr != nil {
				require.ErrorIs(t, err, tt.repoErr)
				assert.Nil(t, got)
				assert.Nil(t, gotCursor)
			} else {
				require.NoError(t, err)
				assert.Equal(t, users, got)
				assert.Equal(t, nextCursor, gotCursor)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetByID(t *testing.T) {
	t.Parallel()

	found := &models.User{ID: "user-1", Email: "user@example.com"}
	repoErr := errors.New("user not found")

	tests := []struct {
		name    string
		id      string
		user    *models.User
		repoErr error
	}{
		{
			name: "returns the repository result",
			id:   "user-1",
			user: found,
		},
		{
			name:    "propagates the repository error",
			id:      "missing",
			repoErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockUserRepository)
			repo.On("GetByID", mock.Anything, tt.id).Return(tt.user, tt.repoErr).Once()
			svc := coreservices.NewUserService(repo, nil, new(internaltests.MockLogger))

			got, err := svc.GetByID(context.Background(), tt.id)

			if tt.repoErr != nil {
				require.ErrorIs(t, err, tt.repoErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.user, got)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("delete failed")

	tests := []struct {
		name    string
		id      string
		repoErr error
	}{
		{
			name: "deletes through the repository",
			id:   "user-1",
		},
		{
			name:    "propagates the repository error",
			id:      "user-2",
			repoErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockUserRepository)
			repo.On("Delete", mock.Anything, tt.id).Return(tt.repoErr).Once()
			svc := coreservices.NewUserService(repo, nil, new(internaltests.MockLogger))

			err := svc.Delete(context.Background(), tt.id)

			if tt.repoErr != nil {
				require.ErrorIs(t, err, tt.repoErr)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_Create_ValidationErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("lookup failed")

	tests := []struct {
		name    string
		email   string
		setup   func(*internaltests.MockUserRepository)
		wantErr error
	}{
		{
			name:    "empty email is rejected before the repository is used",
			email:   "   ",
			wantErr: coreservices.ErrEmailRequired,
		},
		{
			name:  "existing email is rejected",
			email: "jane@example.com",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "jane@example.com").
					Return(&models.User{ID: "user-1", Email: "jane@example.com"}, nil).Once()
			},
			wantErr: coreservices.ErrEmailInUse,
		},
		{
			name:  "lookup failure is treated as no existing user and creation continues",
			email: "new@example.com",
			setup: func(repo *internaltests.MockUserRepository) {
				repo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, repoErr).Once()
				repo.On("Create", mock.Anything, mock.MatchedBy(func(user *models.User) bool {
					return user.Email == "new@example.com"
				})).Return(&models.User{ID: "user-2", Email: "new@example.com"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockUserRepository)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := coreservices.NewUserService(repo, nil, new(internaltests.MockLogger))

			created, err := svc.Create(context.Background(), "Jane", tt.email, false, nil, nil)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, created)
			} else {
				require.NoError(t, err)
				require.NotNil(t, created)
				assert.Equal(t, "new@example.com", created.Email)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserService_Update_RepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("update failed")
	user := &models.User{ID: "user-1", Name: "Jane", Email: "jane@example.com"}

	repo := new(internaltests.MockUserRepository)
	repo.On("Update", mock.Anything, user).Return(nil, repoErr).Once()
	svc := coreservices.NewUserService(repo, nil, new(internaltests.MockLogger))

	updated, err := svc.Update(context.Background(), user)

	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, updated)
	repo.AssertExpectations(t)
}

func TestUserService_Create_RepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("insert failed")

	tests := []struct {
		name      string
		withHooks bool
	}{
		{
			name: "repository error is returned",
		},
		{
			name:      "repository error skips the after hooks",
			withHooks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(internaltests.MockUserRepository)
			repo.On("GetByEmail", mock.Anything, "jane@example.com").Return(nil, nil).Once()
			repo.On("Create", mock.Anything, mock.MatchedBy(func(user *models.User) bool {
				return user.Email == "jane@example.com"
			})).Return(nil, repoErr).Once()

			logger := new(internaltests.RecordingLogger)
			var hooks *models.CoreServiceHooksConfig
			afterHookCalled := false
			if tt.withHooks {
				hooks = newUserHooks()
				hooks.Users.RegisterAfterCreate(func(*models.User) error {
					afterHookCalled = true
					return nil
				})
			}

			svc := coreservices.NewUserService(repo, hooks, logger)

			created, err := svc.Create(context.Background(), "Jane", "jane@example.com", false, nil, nil)

			require.ErrorIs(t, err, repoErr)
			assert.Nil(t, created)
			assert.False(t, afterHookCalled, "after hooks only run once the user is persisted")
			assert.Empty(t, logger.ErrorCalls)
			repo.AssertExpectations(t)
		})
	}
}
