package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func TestVerifyBackupCodeUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		pendingToken string
		trustDevice  bool
		setup        func(t *testing.T, f *testFixture) (code string, ipAddress, userAgent *string)
		assert       func(t *testing.T, result *types.VerifyResult, err error, f *testFixture)
	}{
		{
			name:         "success with trust device",
			pendingToken: "pending-token",
			trustDevice:  true,
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				pending := "pending-token"
				userID := "user-1"
				verificationID := "verif-1"
				ipAddress := "1.2.3.4"
				userAgent := "Mozilla"

				f.expectValidPendingToken(userID, verificationID, pending)
				hashedBackupCodesJSON := `[
					"hash-1",
					"hash-2"
				]`
				f.totpRepo.On("GetByUserID", mock.Anything, userID).Return(&types.TOTPRecord{UserID: userID, BackupCodes: hashedBackupCodesJSON}, nil).Once()
				f.passwordSvc.On("Verify", "my-code", "hash-1").Return(true).Once()
				f.totpRepo.On("CompareAndSwapBackupCodes", mock.Anything, userID, hashedBackupCodesJSON, mock.AnythingOfType("string")).Return(true, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session").Once()
				f.sessionSvc.On("Create", mock.Anything, userID, "hashed-session", &ipAddress, &userAgent, f.globalConfig.Session.ExpiresIn).
					Return(&models.Session{ID: "session-1", UserID: userID}, nil).Once()
				f.verifSvc.On("Delete", mock.Anything, verificationID).Return(nil).Once()
				f.tokenSvc.On("Generate").Return("trusted-device-token", nil).Once()
				f.tokenSvc.On("Hash", "trusted-device-token").Return("hashed-device").Once()
				f.totpRepo.On("CreateTrustedDevice", mock.Anything, userID, "hashed-device", userAgent, mock.AnythingOfType("time.Time")).
					Return(&types.TrustedDevice{ID: "trusted-1", UserID: userID}, nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()

				return "my-code", &ipAddress, &userAgent
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.NoError(t, err)
				require.Equal(t, "session-token", result.SessionToken)
				require.Equal(t, "trusted-device-token", result.TrustedDeviceToken)
				require.NotNil(t, result.Session)
				require.NotNil(t, result.User)
			},
		},
		{
			name:         "returns invalid backup code",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				hashedBackupCodesJSON := `[
					"hash-1"
				]`
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: hashedBackupCodesJSON}, nil).Once()
				f.passwordSvc.On("Verify", "bad", "hash-1").Return(false).Once()
				return "bad", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrInvalidBackupCode)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns invalid pending token",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.tokenSvc.On("Hash", "pending-token").Return("hashed-pending-token").Once()
				f.verifSvc.On("GetByToken", mock.Anything, "hashed-pending-token").Return(nil, nil).Once()
				return "bad", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrInvalidPendingToken)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns not enabled",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				return "bad", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrTOTPNotEnabled)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns invalid backup codes json",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: "not-json"}, nil).Once()
				return "bad", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.Error(t, err)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns invalid backup code when cas fails",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				hashedBackupCodesJSON := `[
					"hash-1"
				]`
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: hashedBackupCodesJSON}, nil).Once()
				f.passwordSvc.On("Verify", "code", "hash-1").Return(true).Once()
				f.totpRepo.On("CompareAndSwapBackupCodes", mock.Anything, "user-1", hashedBackupCodesJSON, mock.AnythingOfType("string")).Return(false, nil).Once()
				return "code", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrInvalidBackupCode)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns session create error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: `[
					"hash-1"
				]`}, nil).Once()
				f.passwordSvc.On("Verify", "code", "hash-1").Return(true).Once()
				f.totpRepo.On("CompareAndSwapBackupCodes", mock.Anything, "user-1", `[
					"hash-1"
				]`, mock.AnythingOfType("string")).Return(true, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session", (*string)(nil), (*string)(nil), f.globalConfig.Session.ExpiresIn).
					Return(nil, errors.New("session create failed")).Once()
				return "code", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "session create failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns user not found",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: `[
					"hash-1"
				]`}, nil).Once()
				f.passwordSvc.On("Verify", "code", "hash-1").Return(true).Once()
				f.totpRepo.On("CompareAndSwapBackupCodes", mock.Anything, "user-1", `[
					"hash-1"
				]`, mock.AnythingOfType("string")).Return(true, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
				return "code", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrUserNotFound)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns verification delete error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: `[
					"hash-1"
				]`}, nil).Once()
				f.passwordSvc.On("Verify", "code", "hash-1").Return(true).Once()
				f.totpRepo.On("CompareAndSwapBackupCodes", mock.Anything, "user-1", `[
					"hash-1"
				]`, mock.AnythingOfType("string")).Return(true, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session", (*string)(nil), (*string)(nil), f.globalConfig.Session.ExpiresIn).
					Return(&models.Session{ID: "session-1", UserID: "user-1"}, nil).Once()
				f.verifSvc.On("Delete", mock.Anything, "verif-1").Return(errors.New("verification delete failed")).Once()
				return "code", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "verification delete failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns repo get error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, errors.New("repo failed")).Once()
				return "code", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "repo failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns compare and swap error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", BackupCodes: `[
					"hash-1"
				]`}, nil).Once()
				f.passwordSvc.On("Verify", "code", "hash-1").Return(true).Once()
				f.totpRepo.On("CompareAndSwapBackupCodes", mock.Anything, "user-1", `[
					"hash-1"
				]`, mock.AnythingOfType("string")).Return(false, errors.New("cas failed")).Once()
				return "code", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "cas failed")
				require.Nil(t, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTestFixture()
			code, ipAddress, userAgent := tc.setup(t, f)

			uc := NewVerifyBackupCodeUseCase(f.globalConfig, f.config, f.logger, f.eventBus, f.tokenSvc, f.sessionSvc, f.userSvc, f.verifSvc, f.backupSvc, f.totpRepo)
			result, err := uc.Verify(context.Background(), tc.pendingToken, code, tc.trustDevice, ipAddress, userAgent)
			tc.assert(t, result, err, f)
		})
	}
}
