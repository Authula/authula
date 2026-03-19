package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func TestVerifyTOTPUseCase(t *testing.T) {
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
				secret := "ABCDEFGHIJKLMNOPQRST"
				code := mustGenerateTOTPCode(t, f.totpSvc, secret)

				f.totpRepo.On("GetByUserID", mock.Anything, userID).Return(&types.TOTPRecord{UserID: userID, Secret: "enc-secret", Enabled: false}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return(secret, nil).Once()
				f.totpRepo.On("SetEnabled", mock.Anything, userID, true).Return(nil).Once()
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

				return code, &ipAddress, &userAgent
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
			name:         "returns invalid code",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret", Enabled: true}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return("ABCDEFGHIJKLMNOPQRST", nil).Once()
				return "123456", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrInvalidTOTPCode)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns invalid pending token",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.tokenSvc.On("Hash", "pending-token").Return("hashed-pending-token").Once()
				f.verifSvc.On("GetByToken", mock.Anything, "hashed-pending-token").Return(nil, nil).Once()
				return "000000", nil, nil
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
				return "000000", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrTOTPNotEnabled)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns decrypt error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret", Enabled: true}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return("", errors.New("decrypt failed")).Once()
				return "000000", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "decrypt failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns invalid verification type",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.tokenSvc.On("Hash", "pending-token").Return("hashed-pending-token").Once()
				verif := &models.Verification{ID: "verif-1", UserID: internaltests.PtrString("user-1"), Type: models.TypeEmailVerification}
				f.verifSvc.On("GetByToken", mock.Anything, "hashed-pending-token").Return(verif, nil).Once()
				return "000000", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrInvalidVerificationType)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns expired pending token",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.tokenSvc.On("Hash", "pending-token").Return("hashed-pending-token").Once()
				verif := &models.Verification{ID: "verif-1", UserID: internaltests.PtrString("user-1"), Type: models.TypeTOTPPendingAuth}
				f.verifSvc.On("GetByToken", mock.Anything, "hashed-pending-token").Return(verif, nil).Once()
				f.verifSvc.On("IsExpired", verif).Return(true).Once()
				return "000000", nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrPendingTokenExpired)
				require.Nil(t, result)
			},
		},
		{
			name:         "returns session create error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				secret := "ABCDEFGHIJKLMNOPQRST"
				code := mustGenerateTOTPCode(t, f.totpSvc, secret)

				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret", Enabled: true}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return(secret, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session", (*string)(nil), (*string)(nil), f.globalConfig.Session.ExpiresIn).
					Return(nil, errors.New("session create failed")).Once()
				return code, nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "session create failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns trusted device create error",
			pendingToken: "pending-token",
			trustDevice:  true,
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				ipAddress := "1.2.3.4"
				userAgent := "Mozilla"
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				secret := "ABCDEFGHIJKLMNOPQRST"
				code := mustGenerateTOTPCode(t, f.totpSvc, secret)

				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret", Enabled: true}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return(secret, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session", &ipAddress, &userAgent, f.globalConfig.Session.ExpiresIn).
					Return(&models.Session{ID: "session-1", UserID: "user-1"}, nil).Once()
				f.verifSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.tokenSvc.On("Generate").Return("device-token", nil).Once()
				f.tokenSvc.On("Hash", "device-token").Return("hashed-device").Once()
				f.totpRepo.On("CreateTrustedDevice", mock.Anything, "user-1", "hashed-device", userAgent, mock.AnythingOfType("time.Time")).
					Return(nil, errors.New("create trusted device failed")).Once()
				return code, &ipAddress, &userAgent
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "create trusted device failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns verification delete error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				secret := "ABCDEFGHIJKLMNOPQRST"
				code := mustGenerateTOTPCode(t, f.totpSvc, secret)

				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret", Enabled: true}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return(secret, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session", (*string)(nil), (*string)(nil), f.globalConfig.Session.ExpiresIn).
					Return(&models.Session{ID: "session-1", UserID: "user-1"}, nil).Once()
				f.verifSvc.On("Delete", mock.Anything, "verif-1").Return(errors.New("verification delete failed")).Once()
				return code, nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "verification delete failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns set enabled error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				secret := "ABCDEFGHIJKLMNOPQRST"
				code := mustGenerateTOTPCode(t, f.totpSvc, secret)

				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret", Enabled: false}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return(secret, nil).Once()
				f.totpRepo.On("SetEnabled", mock.Anything, "user-1", true).Return(errors.New("set enabled failed")).Once()
				return code, nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "set enabled failed")
				require.Nil(t, result)
			},
		},
		{
			name:         "returns user service error",
			pendingToken: "pending-token",
			setup: func(t *testing.T, f *testFixture) (string, *string, *string) {
				f.expectValidPendingToken("user-1", "verif-1", "pending-token")
				secret := "ABCDEFGHIJKLMNOPQRST"
				code := mustGenerateTOTPCode(t, f.totpSvc, secret)

				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Secret: "enc-secret", Enabled: true}, nil).Once()
				f.tokenSvc.On("Decrypt", "enc-secret").Return(secret, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, errors.New("user lookup failed")).Once()
				return code, nil, nil
			},
			assert: func(t *testing.T, result *types.VerifyResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "user lookup failed")
				require.Nil(t, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTestFixture()
			code, ipAddress, userAgent := tc.setup(t, f)

			uc := NewVerifyTOTPUseCase(f.globalConfig, f.config, f.logger, f.eventBus, f.tokenSvc, f.sessionSvc, f.userSvc, f.verifSvc, f.totpSvc, f.totpRepo)
			result, err := uc.Verify(context.Background(), tc.pendingToken, code, tc.trustDevice, ipAddress, userAgent)
			tc.assert(t, result, err, f)
		})
	}
}
