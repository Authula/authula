package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	totptests "github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/usecases"
)

// buildVerifyBackupCodeUseCase builds a VerifyBackupCodeUseCase wired with the given dependencies.
func buildVerifyBackupCodeUseCase(
	tokenSvc *internaltests.MockTokenService,
	verifSvc *internaltests.MockVerificationService,
	sessionSvc *internaltests.MockSessionService,
	userSvc *internaltests.MockUserService,
	repo *totptests.MockTOTPRepo,
	passwordSvc *internaltests.MockPasswordService,
	cfg *types.TOTPPluginConfig,
) *usecases.VerifyBackupCodeUseCase {
	return usecases.NewVerifyBackupCodeUseCase(
		&models.Config{Session: models.SessionConfig{ExpiresIn: 24 * time.Hour}},
		cfg,
		&internaltests.MockLogger{},
		totptests.NewNoopEventBus(),
		tokenSvc,
		sessionSvc,
		userSvc,
		verifSvc,
		services.NewBackupCodeService(10, passwordSvc),
		repo,
	)
}

// setupValidVerification configures mock services so that resolvePendingToken succeeds
// and returns userID "user-1". It returns the verification record for further setup.
func setupValidVerification(
	tokenSvc *internaltests.MockTokenService,
	verifSvc *internaltests.MockVerificationService,
	rawToken, userID string,
) *models.Verification {
	hashedToken := "hashed-" + rawToken
	tokenSvc.On("Hash", rawToken).Return(hashedToken)
	verif := &models.Verification{
		ID:     "verif-id",
		UserID: internaltests.PtrString(userID),
		Type:   models.TypeTOTPPendingAuth,
	}
	verifSvc.On("GetByToken", mock.Anything, hashedToken).Return(verif, nil)
	verifSvc.On("IsExpired", verif).Return(false)
	return verif
}

func mustGenerateTOTPSecret(t *testing.T) string {
	t.Helper()
	totpSvc := services.NewTOTPService(6, 30)
	plainSecret, err := totpSvc.GenerateSecret()
	require.NoError(t, err)
	return plainSecret
}

func mustGenerateTOTPCode(t *testing.T, plainSecret string) string {
	t.Helper()
	totpSvc := services.NewTOTPService(6, 30)
	validCode, err := totpSvc.GenerateCode(plainSecret, time.Now().UTC())
	require.NoError(t, err)
	return validCode
}

type GenerateBackupCodesHandlerSuite struct {
	suite.Suite
}

type generateBackupCodesFixture struct {
	repo        *totptests.MockTOTPRepo
	passwordSvc *internaltests.MockPasswordService
	backupSvc   *services.BackupCodeService
}

type generateBackupCodesCase struct {
	name           string
	userID         *string
	prepare        func(m *generateBackupCodesFixture)
	expectedStatus int
	checkResponse  func(t *testing.T, reqCtx *models.RequestContext)
}

func TestGenerateBackupCodesHandlerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GenerateBackupCodesHandlerSuite))
}

func (s *GenerateBackupCodesHandlerSuite) TestGenerateBackupCodesHandler_Table() {
	uid := "user-1"
	tests := []generateBackupCodesCase{
		{
			name:           "unauthorized",
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "usecase_not_enabled",
			userID: internaltests.PtrString(uid),
			prepare: func(m *generateBackupCodesFixture) {
				m.repo.On("GetByUserID", mock.Anything, uid).Return(nil, nil)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				assert.Contains(t, string(reqCtx.ResponseBody), constants.ErrTOTPNotEnabled.Error())
			},
		},
		{
			name:   "success",
			userID: internaltests.PtrString(uid),
			prepare: func(m *generateBackupCodesFixture) {
				m.repo.On("GetByUserID", mock.Anything, uid).Return(&types.TOTPRecord{
					UserID:      uid,
					BackupCodes: `[]`,
					Enabled:     true,
				}, nil)
				m.repo.On("UpdateBackupCodes", mock.Anything, uid, mock.AnythingOfType("string")).Return(nil)
				m.passwordSvc.On("Hash", mock.Anything).Return("hashed-code", nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				var resp types.GenerateBackupCodesResponse
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &resp))
				assert.Len(t, resp.BackupCodes, 3)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			t := s.T()
			m := &generateBackupCodesFixture{
				repo:        &totptests.MockTOTPRepo{},
				passwordSvc: &internaltests.MockPasswordService{},
			}
			m.backupSvc = services.NewBackupCodeService(3, m.passwordSvc)

			if tt.prepare != nil {
				tt.prepare(m)
			}

			uc := usecases.NewGenerateBackupCodesUseCase(m.backupSvc, m.repo)
			h := &GenerateBackupCodesHandler{UseCase: uc}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/totp/generate-backup-codes", nil, tt.userID)
			h.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
		})
	}
}
