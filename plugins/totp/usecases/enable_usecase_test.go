package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func TestEnableUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
		issuer string
		setup  func(*testFixture)
		assert func(t *testing.T, result *types.EnableResult, err error, f *testFixture)
	}{
		{
			name:   "existing disabled record allows retry",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.config.SkipVerificationOnEnable = false
				uid := "user-1"

				f.totpRepo.On("GetByUserID", mock.Anything, uid).Return(&types.TOTPRecord{UserID: uid, Enabled: false}, nil).Once()
				f.totpRepo.On("DeleteByUserID", mock.Anything, uid).Return(nil).Once()
				f.totpRepo.On("Create", mock.Anything, uid, "enc-secret", mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: uid}, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, uid).Return(&models.User{ID: uid, Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("hashed-backup", nil).Times(2)
				f.tokenSvc.On("Generate").Return("pending-token", nil).Once()
				f.tokenSvc.On("Hash", "pending-token").Return("hashed-pending").Once()
				f.verifSvc.On("Create", mock.Anything, uid, "hashed-pending", models.TypeTOTPPendingAuth, uid, 5*time.Minute).
					Return(&models.Verification{ID: "verif-1"}, nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, "pending-token", result.PendingToken)
				require.Len(t, result.BackupCodes, 2)
				require.NotEmpty(t, result.TotpURI)
			},
		},
		{
			name:   "existing enabled record returns already enabled",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.TOTPRecord{UserID: "user-1", Enabled: true}, nil).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrTOTPAlreadyEnabled)
				require.Nil(t, result)
				f.totpRepo.AssertNotCalled(t, "DeleteByUserID", mock.Anything, "user-1")
				f.totpRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:   "skip verification on enable sets enabled",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.config.SkipVerificationOnEnable = true
				uid := "user-1"

				f.totpRepo.On("GetByUserID", mock.Anything, uid).Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, uid).Return(&models.User{ID: uid, Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("DeleteByUserID", mock.Anything, uid).Return(nil).Once()
				f.totpRepo.On("Create", mock.Anything, uid, "enc-secret", mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: uid}, nil).Once()
				f.totpRepo.On("SetEnabled", mock.Anything, uid, true).Return(nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Empty(t, result.PendingToken)
				f.verifSvc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.tokenSvc.AssertNotCalled(t, "Generate")
			},
		},
		{
			name:   "returns user not found",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorIs(t, err, constants.ErrUserNotFound)
				require.Nil(t, result)
			},
		},
		{
			name:   "returns encrypt error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("", errors.New("encrypt failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "encrypt failed")
				require.Nil(t, result)
			},
		},
		{
			name:   "returns backup hash error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("", errors.New("hash failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "hash failed")
				require.Nil(t, result)
			},
		},
		{
			name:   "uses provided issuer",
			userID: "user-1",
			issuer: "CustomIssuer",
			setup: func(f *testFixture) {
				f.config.SkipVerificationOnEnable = true
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.totpRepo.On("Create", mock.Anything, "user-1", "enc-secret", mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.totpRepo.On("SetEnabled", mock.Anything, "user-1", true).Return(nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Contains(t, result.TotpURI, "issuer=CustomIssuer")
			},
		},
		{
			name:   "returns delete error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(errors.New("delete failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "delete failed")
				require.Nil(t, result)
			},
		},
		{
			name:   "returns create error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.totpRepo.On("Create", mock.Anything, "user-1", "enc-secret", mock.AnythingOfType("string")).Return(nil, errors.New("create failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "create failed")
				require.Nil(t, result)
			},
		},
		{
			name:   "returns verification create error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.totpRepo.On("Create", mock.Anything, "user-1", "enc-secret", mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.tokenSvc.On("Generate").Return("pending-token", nil).Once()
				f.tokenSvc.On("Hash", "pending-token").Return("hashed-pending").Once()
				f.verifSvc.On("Create", mock.Anything, "user-1", "hashed-pending", models.TypeTOTPPendingAuth, "user-1", f.config.PendingTokenExpiry).
					Return(nil, errors.New("verification create failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "verification create failed")
				require.Nil(t, result)
			},
		},
		{
			name:   "returns get by user id error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, errors.New("repo failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "repo failed")
				require.Nil(t, result)
			},
		},
		{
			name:   "returns set enabled error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.config.SkipVerificationOnEnable = true
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.totpRepo.On("Create", mock.Anything, "user-1", "enc-secret", mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.totpRepo.On("SetEnabled", mock.Anything, "user-1", true).Return(errors.New("set enabled failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "set enabled failed")
				require.Nil(t, result)
			},
		},
		{
			name:   "returns pending token generate error",
			userID: "user-1",
			setup: func(f *testFixture) {
				f.config.SkipVerificationOnEnable = false
				f.totpRepo.On("GetByUserID", mock.Anything, "user-1").Return(nil, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil).Once()
				f.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Once()
				f.passwordSvc.On("Hash", mock.Anything).Return("h", nil).Times(2)
				f.totpRepo.On("DeleteByUserID", mock.Anything, "user-1").Return(nil).Once()
				f.totpRepo.On("Create", mock.Anything, "user-1", "enc-secret", mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: "user-1"}, nil).Once()
				f.tokenSvc.On("Generate").Return("", errors.New("generate failed")).Once()
			},
			assert: func(t *testing.T, result *types.EnableResult, err error, f *testFixture) {
				require.ErrorContains(t, err, "generate failed")
				require.Nil(t, result)
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

			uc := NewEnableUseCase(f.config, f.logger, f.eventBus, f.userSvc, f.tokenSvc, f.verifSvc, f.totpSvc, f.backupSvc, f.totpRepo)
			result, err := uc.Enable(context.Background(), tc.userID, tc.issuer)
			tc.assert(t, result, err, f)
		})
	}
}
