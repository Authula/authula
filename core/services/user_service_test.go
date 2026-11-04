package services_test

import (
	"context"
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
