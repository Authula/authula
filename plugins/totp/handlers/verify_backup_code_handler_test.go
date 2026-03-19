package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	totptests "github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

type VerifyBackupCodeHandlerSuite struct {
	suite.Suite
}

type verifyBackupFixture struct {
	pluginCfg   *types.TOTPPluginConfig
	tokenSvc    *internaltests.MockTokenService
	verifSvc    *internaltests.MockVerificationService
	sessionSvc  *internaltests.MockSessionService
	userSvc     *internaltests.MockUserService
	repo        *totptests.MockTOTPRepo
	passwordSvc *internaltests.MockPasswordService
	pendingRaw  string
	uid         string
	requestBody []byte
	withCookie  bool
}

func newVerifyBackupFixture() *verifyBackupFixture {
	return &verifyBackupFixture{
		pluginCfg:   totptests.NewPluginConfig(),
		tokenSvc:    &internaltests.MockTokenService{},
		verifSvc:    &internaltests.MockVerificationService{},
		sessionSvc:  &internaltests.MockSessionService{},
		userSvc:     &internaltests.MockUserService{},
		repo:        &totptests.MockTOTPRepo{},
		passwordSvc: &internaltests.MockPasswordService{},
		pendingRaw:  "pending-tok",
		uid:         "user-1",
	}
}

func (m *verifyBackupFixture) newRequest(t *testing.T) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/totp/verify-backup-code", m.requestBody, internaltests.PtrString(m.uid))
	if m.withCookie {
		req.AddCookie(&http.Cookie{Name: constants.CookieTOTPPending, Value: m.pendingRaw})
	}
	return req, w, reqCtx
}

type verifyBackupCase struct {
	name           string
	prepare        func(m *verifyBackupFixture)
	expectedStatus int
	checkResponse  func(t *testing.T, w *httptest.ResponseRecorder, reqCtx *models.RequestContext)
}

func TestVerifyBackupCodeHandlerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VerifyBackupCodeHandlerSuite))
}

func (s *VerifyBackupCodeHandlerSuite) TestVerifyBackupCodeHandler_Table() {
	tests := []verifyBackupCase{
		{
			name:           "missing_pending_cookie",
			prepare:        func(m *verifyBackupFixture) { m.requestBody = []byte(`{"code":"abc"}`) },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid_body",
			prepare: func(m *verifyBackupFixture) {
				m.requestBody = []byte("bad")
				m.withCookie = true
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid_pending_token",
			prepare: func(m *verifyBackupFixture) {
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyBackupCodeRequest{Code: "ABCD-1234"})
				m.tokenSvc.On("Hash", m.pendingRaw).Return("hashed-tok")
				m.verifSvc.On("GetByToken", mock.Anything, "hashed-tok").Return(nil, nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "totp_not_enabled",
			prepare: func(m *verifyBackupFixture) {
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyBackupCodeRequest{Code: "ABCD-1234"})
				setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(nil, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				assert.Contains(t, string(reqCtx.ResponseBody), constants.ErrTOTPNotEnabled.Error())
			},
		},
		{
			name: "invalid_backup_code",
			prepare: func(m *verifyBackupFixture) {
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyBackupCodeRequest{Code: "WRONG-CODE"})
				m.passwordSvc.On("Verify", mock.Anything, mock.Anything).Return(false)
				setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(&types.TOTPRecord{UserID: m.uid, BackupCodes: `["hashed-backup-code"]`, Enabled: true}, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				assert.Contains(t, string(reqCtx.ResponseBody), constants.ErrInvalidBackupCode.Error())
			},
		},
		{
			name: "success",
			prepare: func(m *verifyBackupFixture) {
				const hashedCode = "hashed-valid-backup"
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyBackupCodeRequest{Code: "VALID-BACKUP"})
				m.passwordSvc.On("Verify", "valid-backup", hashedCode).Return(true)

				verif := setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.verifSvc.On("Delete", mock.Anything, verif.ID).Return(nil)
				m.tokenSvc.On("Generate").Return("session-token", nil)
				m.tokenSvc.On("Hash", "session-token").Return("hashed-session-token")

				backupCodesJSON, _ := json.Marshal([]string{hashedCode})
				emptyCodesJSON, _ := json.Marshal([]string{})
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(&types.TOTPRecord{UserID: m.uid, BackupCodes: string(backupCodesJSON), Enabled: true}, nil)
				m.repo.On("CompareAndSwapBackupCodes", mock.Anything, m.uid, string(backupCodesJSON), string(emptyCodesJSON)).Return(true, nil)

				m.sessionSvc.On("Create", mock.Anything, m.uid, "hashed-session-token", mock.Anything, mock.Anything, mock.Anything).
					Return(&models.Session{ID: "sess-id", UserID: m.uid}, nil)
				m.userSvc.On("GetByID", mock.Anything, m.uid).Return(&models.User{ID: m.uid, Email: "user@example.com"}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				assert.Equal(t, true, reqCtx.Values[models.ContextAuthSuccess.String()])
				assert.Equal(t, "sess-id", reqCtx.Values[models.ContextSessionID.String()])
				cleared := totptests.CookieFromRecorder(w, constants.CookieTOTPPending)
				require.NotNil(t, cleared)
				assert.Equal(t, -1, cleared.MaxAge)
				assert.Nil(t, totptests.CookieFromRecorder(w, constants.CookieTOTPTrusted))
			},
		},
		{
			name: "success_with_trusted_device",
			prepare: func(m *verifyBackupFixture) {
				const hashedCode = "hashed-valid-backup"
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyBackupCodeRequest{Code: "VALID-BACKUP", TrustDevice: true})
				m.passwordSvc.On("Verify", "valid-backup", hashedCode).Return(true)

				verif := setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.verifSvc.On("Delete", mock.Anything, verif.ID).Return(nil)
				m.tokenSvc.On("Generate").Return("session-token", nil).Once()
				m.tokenSvc.On("Generate").Return("trusted-tok", nil).Once()
				m.tokenSvc.On("Hash", "session-token").Return("hashed-session-token")
				m.tokenSvc.On("Hash", "trusted-tok").Return("hashed-trusted-tok")

				backupCodesJSON, _ := json.Marshal([]string{hashedCode})
				emptyCodesJSON, _ := json.Marshal([]string{})
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(&types.TOTPRecord{UserID: m.uid, BackupCodes: string(backupCodesJSON), Enabled: true}, nil)
				m.repo.On("CompareAndSwapBackupCodes", mock.Anything, m.uid, string(backupCodesJSON), string(emptyCodesJSON)).Return(true, nil)
				m.repo.On("CreateTrustedDevice", mock.Anything, m.uid, "hashed-trusted-tok", mock.Anything, mock.Anything).
					Return(&types.TrustedDevice{ID: "td-1"}, nil)

				m.sessionSvc.On("Create", mock.Anything, m.uid, "hashed-session-token", mock.Anything, mock.Anything, mock.Anything).
					Return(&models.Session{ID: "sess-id", UserID: m.uid}, nil)
				m.userSvc.On("GetByID", mock.Anything, m.uid).Return(&models.User{ID: m.uid, Email: "user@example.com"}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, _ *models.RequestContext) {
				trusted := totptests.CookieFromRecorder(w, constants.CookieTOTPTrusted)
				require.NotNil(t, trusted, "trusted device cookie should be set")
				assert.Equal(t, "trusted-tok", trusted.Value)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			t := s.T()
			m := newVerifyBackupFixture()

			if tt.prepare != nil {
				tt.prepare(m)
			}

			uc := buildVerifyBackupCodeUseCase(m.tokenSvc, m.verifSvc, m.sessionSvc, m.userSvc, m.repo, m.passwordSvc, m.pluginCfg)
			h := &VerifyBackupCodeHandler{UseCase: uc, PluginConfig: m.pluginCfg}
			req, w, reqCtx := m.newRequest(t)
			h.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w, reqCtx)
			}
		})
	}
}
