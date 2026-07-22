package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/services"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestRequestEmailChangeUseCaseHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name: "before request email change hook fires with user",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					EmailChange: &types.EmailChangeServiceHooksConfig{
						BeforeRequestEmailChange: func(ctx context.Context, user *models.User) error {
							require.Equal(t, "user-1", user.ID)
							return nil
						},
					},
				}, f.logger)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "old@example.com"}, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
				f.tokenSvc.On("Generate").Return("token-123", nil)
				f.tokenSvc.On("Hash", mock.Anything).Return("hashed-token", nil)
				f.verificationSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&models.Verification{ID: "ver-1"}, nil)
				f.mailerSvc.On("SendEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "before request email change hook rejects",
			setup: func(f *emailPasswordTestFixture) {
				f.hooksExecutor = services.NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
					EmailChange: &types.EmailChangeServiceHooksConfig{
						BeforeRequestEmailChange: func(ctx context.Context, user *models.User) error {
							return errors.New("email change not allowed")
						},
					},
				}, f.logger)
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "old@example.com"}, nil)
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
			},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, "email change not allowed", err.Error())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			tc.setup(f)
			err := f.requestEmailChangeUseCase().RequestChange(testRequestContext(), "user-1", "new@example.com", nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
		})
	}
}
