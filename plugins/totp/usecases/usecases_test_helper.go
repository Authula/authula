package usecases

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	totptests "github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

type testFixture struct {
	globalConfig *models.Config
	config       *types.TOTPPluginConfig
	logger       *internaltests.MockLogger
	eventBus     *internaltests.MockEventBus
	totpRepo     *totptests.MockTOTPRepo
	userSvc      *internaltests.MockUserService
	tokenSvc     *internaltests.MockTokenService
	sessionSvc   *internaltests.MockSessionService
	verifSvc     *internaltests.MockVerificationService
	passwordSvc  *internaltests.MockPasswordService
	totpSvc      *services.TOTPService
	backupSvc    *services.BackupCodeService
}

func newTestFixture() *testFixture {
	passwordSvc := &internaltests.MockPasswordService{}
	return &testFixture{
		globalConfig: &models.Config{
			Session: models.SessionConfig{ExpiresIn: 24 * time.Hour},
		},
		config: &types.TOTPPluginConfig{
			PendingTokenExpiry:    5 * time.Minute,
			TrustedDeviceDuration: 24 * time.Hour,
		},
		logger:      &internaltests.MockLogger{},
		eventBus:    &internaltests.MockEventBus{},
		userSvc:     &internaltests.MockUserService{},
		tokenSvc:    &internaltests.MockTokenService{},
		sessionSvc:  &internaltests.MockSessionService{},
		verifSvc:    &internaltests.MockVerificationService{},
		passwordSvc: passwordSvc,
		totpSvc:     services.NewTOTPService(6, 30),
		backupSvc:   services.NewBackupCodeService(2, passwordSvc),
		totpRepo:    &totptests.MockTOTPRepo{},
	}
}

func mustGenerateTOTPCode(t *testing.T, svc *services.TOTPService, secret string) string {
	t.Helper()

	code, err := svc.GenerateCode(secret, time.Now().UTC())
	require.NoError(t, err)
	return code
}

func (f *testFixture) expectValidPendingToken(userID, verificationID, pendingToken string) {
	f.tokenSvc.On("Hash", pendingToken).Return("hashed-" + pendingToken).Once()
	verif := &models.Verification{ID: verificationID, UserID: internaltests.PtrString(userID), Type: models.TypeTOTPPendingAuth}
	f.verifSvc.On("GetByToken", mock.Anything, "hashed-"+pendingToken).Return(verif, nil).Once()
	f.verifSvc.On("IsExpired", verif).Return(false).Once()
}
