package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func TestDisableUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
		setup  func(*testFixture)
		assert func(t *testing.T, err error, f *testFixture)
	}{
		{
			name:   "uses repository",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.totpRepo.On("DeleteTrustedDevicesByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, err error, f *testFixture) {
				require.NoError(t, err)
				f.totpRepo.AssertExpectations(t)
			},
		},
		{
			name:   "returns not enabled when missing",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrTOTPNotEnabled)
			},
		},
		{
			name:   "returns delete error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(errors.New("delete failed")).Once()
			},
			assert: func(t *testing.T, err error, f *testFixture) {
				require.ErrorContains(t, err, "delete failed")
				f.totpRepo.AssertNotCalled(t, "DeleteTrustedDevicesByUserID", mock.Anything, "user-1")
			},
		},
		{
			name:   "returns delete trusted devices error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.totpRepo.On("DeleteTrustedDevicesByUserID", mock.Anything, "user-1").Return(errors.New("trusted delete failed")).Once()
			},
			assert: func(t *testing.T, err error, f *testFixture) {
				require.ErrorContains(t, err, "trusted delete failed")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}

			uc := NewDisableUseCase(f.logger, f.eventBus, f.totpRepo)
			err := uc.Disable(context.Background(), tc.userID)
			tc.assert(t, err, f)
		})
	}
}
