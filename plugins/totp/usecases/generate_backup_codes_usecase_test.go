package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func TestGenerateBackupCodesUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
		setup  func(*testFixture)
		assert func(t *testing.T, codes []string, f *testFixture, err error)
	}{
		{
			name:   "updates repository",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: "[]"}, nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("UpdateBackupCodes", mock.Anything, "user-1", mock.AnythingOfType("string")).Return(nil).Once()
			},
			assert: func(t *testing.T, codes []string, f *testFixture, err error) {
				require.NoError(t, err)
				require.Len(t, codes, 2)

				var stored []string
				updateArgs := f.totpRepo.Calls[len(f.totpRepo.Calls)-1].Arguments
				require.NoError(t, json.Unmarshal([]byte(updateArgs.Get(2).(string)), &stored))
				require.Len(t, stored, 2)
				f.passwordSvc.AssertExpectations(t)
				f.totpRepo.AssertExpectations(t)
			},
		},
		{
			name:   "returns not enabled",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, codes []string, f *testFixture, err error) {
				require.ErrorIs(t, err, constants.ErrTOTPNotEnabled)
				require.Nil(t, codes)
			},
		},
		{
			name:   "returns hash error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("", errors.New("hash failed")).Once()
			},
			assert: func(t *testing.T, codes []string, f *testFixture, err error) {
				require.ErrorContains(t, err, "hash failed")
				require.Nil(t, codes)
				f.totpRepo.AssertNotCalled(t, "UpdateBackupCodes", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:   "returns update error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("UpdateBackupCodes", mock.Anything, "user-1", mock.AnythingOfType("string")).Return(errors.New("update failed")).Once()
			},
			assert: func(t *testing.T, codes []string, f *testFixture, err error) {
				require.ErrorContains(t, err, "update failed")
				require.Nil(t, codes)
			},
		},
		{
			name:   "returns get by user id error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, errors.New("repo failed")).Once()
			},
			assert: func(t *testing.T, codes []string, f *testFixture, err error) {
				require.ErrorContains(t, err, "repo failed")
				require.Nil(t, codes)
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

			uc := NewGenerateBackupCodesUseCase(f.backupSvc, f.totpRepo)
			codes, err := uc.Generate(context.Background(), tc.userID)
			tc.assert(t, codes, f, err)
		})
	}
}
