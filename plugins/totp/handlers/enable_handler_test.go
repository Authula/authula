package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

type EnableHandlerSuite struct {
	suite.Suite
}

type enableHandlerFixture struct {
	config   *types.TOTPPluginConfig
	logger   *internaltests.MockLogger
	eventBus *internaltests.MockEventBus
	userSvc  *internaltests.MockUserService
	tokenSvc *internaltests.MockTokenService
	verifSvc *internaltests.MockVerificationService
	password *internaltests.MockPasswordService
	repo     *totptests.MockTOTPRepo
}

type enableHandlerTestCase struct {
	name           string
	userID         *string
	body           []byte
	prepare        func(m *enableHandlerFixture)
	expectedStatus int
	checkResponse  func(t *testing.T, w *httptest.ResponseRecorder, reqCtx *models.RequestContext)
}

func TestEnableHandlerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(EnableHandlerSuite))
}

func (s *EnableHandlerSuite) TestEnableHandler_Table() {
	uid := "user-1"
	tests := []enableHandlerTestCase{
		{
			name:           "unauthenticated",
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "usecase_already_enabled",
			userID: internaltests.PtrString(uid),
			prepare: func(m *enableHandlerFixture) {
				m.resetRepoExpectations()
				m.repo.On("GetByUserID", mock.Anything, uid).Return(&types.TOTPRecord{UserID: uid, Enabled: true}, nil)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, _ *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				assert.Contains(t, string(reqCtx.ResponseBody), constants.ErrTOTPAlreadyEnabled.Error())
			},
		},
		{
			name:   "existing_but_not_enabled_allows_retry",
			userID: internaltests.PtrString(uid),
			prepare: func(m *enableHandlerFixture) {
				m.resetRepoExpectations()
				m.repo.On("GetByUserID", mock.Anything, uid).Return(&types.TOTPRecord{UserID: uid, Enabled: false}, nil)
				m.repo.On("DeleteByUserID", mock.Anything, uid).Return(nil)
				m.repo.On("Create", mock.Anything, uid, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: uid}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, _ *models.RequestContext) {
				pendingCookie := totptests.CookieFromRecorder(w, constants.CookieTOTPPending)
				require.NotNil(t, pendingCookie, "pending cookie should be set")
				assert.Equal(t, "pending-token-value", pendingCookie.Value)
			},
		},
		{
			name:   "success_skip_verification",
			userID: internaltests.PtrString(uid),
			prepare: func(m *enableHandlerFixture) {
				m.config.SkipVerificationOnEnable = true
				m.resetRepoExpectations()
				m.repo.On("GetByUserID", mock.Anything, uid).Return(nil, nil)
				m.repo.On("DeleteByUserID", mock.Anything, uid).Return(nil)
				m.repo.On("Create", mock.Anything, uid, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&types.TOTPRecord{UserID: uid}, nil)
				m.repo.On("SetEnabled", mock.Anything, uid, true).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				var resp types.EnableResponse
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &resp))
				assert.Contains(t, resp.TotpURI, "otpauth://totp/")
				assert.Len(t, resp.BackupCodes, 2)
				assert.Nil(t, totptests.CookieFromRecorder(w, constants.CookieTOTPPending))
			},
		},
		{
			name:           "success_with_pending_token",
			userID:         internaltests.PtrString(uid),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				pendingCookie := totptests.CookieFromRecorder(w, constants.CookieTOTPPending)
				require.NotNil(t, pendingCookie, "pending cookie should be set")
				assert.Equal(t, "pending-token-value", pendingCookie.Value)

				var resp types.EnableResponse
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &resp))
				assert.Contains(t, resp.TotpURI, "otpauth://totp/")
				assert.Len(t, resp.BackupCodes, 2)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			t := s.T()
			m := newEnableHandlerFixture()
			m.setupHappyPath(uid)

			if tt.prepare != nil {
				tt.prepare(m)
			}

			h := m.buildHandler()
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/totp/enable", tt.body, tt.userID)
			h.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w, reqCtx)
			}
		})
	}
}

func newEnableHandlerFixture() *enableHandlerFixture {
	return &enableHandlerFixture{
		config: &types.TOTPPluginConfig{
			SkipVerificationOnEnable: false,
			PendingTokenExpiry:       5 * time.Minute,
			SecureCookie:             false,
			SameSite:                 "lax",
		},
		logger:   &internaltests.MockLogger{},
		repo:     &totptests.MockTOTPRepo{},
		userSvc:  &internaltests.MockUserService{},
		tokenSvc: &internaltests.MockTokenService{},
		verifSvc: &internaltests.MockVerificationService{},
		eventBus: &internaltests.MockEventBus{},
		password: &internaltests.MockPasswordService{},
	}
}

func (m *enableHandlerFixture) setupHappyPath(uid string) {
	m.repo.On("GetByUserID", mock.Anything, uid).Return(nil, nil).Maybe()
	m.repo.On("DeleteByUserID", mock.Anything, uid).Return(nil).Maybe()
	m.repo.On("Create", mock.Anything, uid, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&types.TOTPRecord{UserID: uid}, nil).Maybe()

	m.userSvc.On("GetByID", mock.Anything, uid).Return(&models.User{ID: uid, Email: "user@example.com"}, nil).Maybe()

	m.tokenSvc.On("Encrypt", mock.AnythingOfType("string")).Return("enc-secret", nil).Maybe()
	m.tokenSvc.On("Generate").Return("pending-token-value", nil).Maybe()
	m.tokenSvc.On("Hash", "pending-token-value").Return("hashed-pending").Maybe()

	m.password.On("Hash", mock.Anything).Return("hashed-backup", nil).Maybe()

	m.verifSvc.On("Create", mock.Anything, uid, "hashed-pending", models.TypeTOTPPendingAuth, uid, 5*time.Minute).
		Return(&models.Verification{ID: "v-1"}, nil).Maybe()

	m.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
}

func (m *enableHandlerFixture) buildHandler() *EnableHandler {
	totpSvc := services.NewTOTPService(6, 30)
	backupSvc := services.NewBackupCodeService(2, m.password)

	if m.config.SkipVerificationOnEnable {
		uc := usecases.NewEnableUseCase(m.config, m.logger, m.eventBus, m.userSvc, m.tokenSvc, nil, totpSvc, backupSvc, m.repo)
		return &EnableHandler{GlobalConfig: &models.Config{AppName: "MyApp"}, PluginConfig: m.config, UseCase: uc}
	}

	uc := usecases.NewEnableUseCase(m.config, m.logger, m.eventBus, m.userSvc, m.tokenSvc, m.verifSvc, totpSvc, backupSvc, m.repo)
	return &EnableHandler{GlobalConfig: &models.Config{AppName: "MyApp"}, PluginConfig: m.config, UseCase: uc}
}

func (m *enableHandlerFixture) resetRepoExpectations() {
	m.repo.ExpectedCalls = nil
}
