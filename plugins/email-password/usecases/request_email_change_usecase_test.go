package usecases

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
)

func TestRequestEmailChangeUseCase_RequestChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name: "missing user returns error",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.ErrorIs(t, err, constants.ErrUserNotFound) },
		},
		{
			name: "creates email change verification",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil).Once()
				f.tokenSvc.On("Generate").Return("change-token", nil).Once()
				f.tokenSvc.On("Hash", "change-token").Return("hashed-token").Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailResetRequest, "new@example.com", time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "new@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			err := f.requestEmailChangeUseCase().RequestChange(testRequestContext(), "user-1", "new@example.com", nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
