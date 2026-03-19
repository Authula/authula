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

func buildVerifyTOTPUseCase(
	cfg *types.TOTPPluginConfig,
	tokenSvc *internaltests.MockTokenService,
	verifSvc *internaltests.MockVerificationService,
	sessionSvc *internaltests.MockSessionService,
	userSvc *internaltests.MockUserService,
	repo *totptests.MockTOTPRepo,
) *usecases.VerifyTOTPUseCase {
	return usecases.NewVerifyTOTPUseCase(
		&models.Config{Session: models.SessionConfig{ExpiresIn: 24 * time.Hour}},
		cfg,
		&internaltests.MockLogger{},
		totptests.NewNoopEventBus(),
		tokenSvc,
		sessionSvc,
		userSvc,
		verifSvc,
		services.NewTOTPService(6, 30),
		repo,
	)
}

func newVerifyTOTPFixture() *verifyTOTPFixture {
	return &verifyTOTPFixture{
		pluginCfg:  totptests.NewPluginConfig(),
		tokenSvc:   &internaltests.MockTokenService{},
		verifSvc:   &internaltests.MockVerificationService{},
		sessionSvc: &internaltests.MockSessionService{},
		userSvc:    &internaltests.MockUserService{},
		repo:       &totptests.MockTOTPRepo{},
		pendingRaw: "pending-token",
		uid:        "user-1",
	}
}

type VerifyTOTPHandlerSuite struct {
	suite.Suite
}

type verifyTOTPFixture struct {
	pluginCfg   *types.TOTPPluginConfig
	tokenSvc    *internaltests.MockTokenService
	verifSvc    *internaltests.MockVerificationService
	sessionSvc  *internaltests.MockSessionService
	userSvc     *internaltests.MockUserService
	repo        *totptests.MockTOTPRepo
	pendingRaw  string
	uid         string
	requestBody []byte
	withCookie  bool
}

func (m *verifyTOTPFixture) newRequest(t *testing.T) (*http.Request, *httptest.ResponseRecorder, *models.RequestContext) {
	req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/totp/verify", m.requestBody, internaltests.PtrString(m.uid))
	if m.withCookie {
		req.AddCookie(&http.Cookie{Name: constants.CookieTOTPPending, Value: m.pendingRaw})
	}
	return req, w, reqCtx
}

type verifyTOTPCase struct {
	name           string
	prepare        func(m *verifyTOTPFixture)
	expectedStatus int
	checkResponse  func(t *testing.T, w *httptest.ResponseRecorder, reqCtx *models.RequestContext)
}

func TestVerifyTOTPHandlerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VerifyTOTPHandlerSuite))
}

func (s *VerifyTOTPHandlerSuite) TestVerifyTOTPHandler_Table() {
	tests := []verifyTOTPCase{
		{
			name:           "missing_pending_cookie",
			prepare:        func(m *verifyTOTPFixture) { m.requestBody = []byte(`{"code":"123456"}`) },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid_body",
			prepare: func(m *verifyTOTPFixture) {
				m.requestBody = []byte("bad")
				m.withCookie = true
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid_pending_token",
			prepare: func(m *verifyTOTPFixture) {
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyTOTPRequest{Code: "123456"})
				m.tokenSvc.On("Hash", m.pendingRaw).Return("hashed-token")
				m.verifSvc.On("GetByToken", mock.Anything, "hashed-token").Return(nil, nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "totp_not_enabled",
			prepare: func(m *verifyTOTPFixture) {
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyTOTPRequest{Code: "123456"})
				setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(nil, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				assert.Contains(t, string(reqCtx.ResponseBody), constants.ErrTOTPNotEnabled.Error())
			},
		},
		{
			name: "invalid_totp_code",
			prepare: func(m *verifyTOTPFixture) {
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyTOTPRequest{Code: "000000"})
				plainSecret := mustGenerateTOTPSecret(s.T())

				setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.tokenSvc.On("Decrypt", "enc-secret").Return(plainSecret, nil)
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(&types.TOTPRecord{UserID: m.uid, Secret: "enc-secret", Enabled: true}, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				assert.Contains(t, string(reqCtx.ResponseBody), constants.ErrInvalidTOTPCode.Error())
			},
		},
		{
			name: "generic_usecase_error",
			prepare: func(m *verifyTOTPFixture) {
				m.withCookie = true
				m.requestBody, _ = json.Marshal(types.VerifyTOTPRequest{Code: "123456"})
				setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.tokenSvc.On("Decrypt", "enc-secret").Return(nil, assert.AnError)
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(&types.TOTPRecord{UserID: m.uid, Secret: "enc-secret", Enabled: true}, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "success",
			prepare: func(m *verifyTOTPFixture) {
				m.withCookie = true
				plainSecret := mustGenerateTOTPSecret(s.T())
				validCode := mustGenerateTOTPCode(s.T(), plainSecret)
				m.requestBody, _ = json.Marshal(types.VerifyTOTPRequest{Code: validCode})

				verif := setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.verifSvc.On("Delete", mock.Anything, verif.ID).Return(nil)
				m.tokenSvc.On("Decrypt", "enc-secret").Return(plainSecret, nil)
				m.tokenSvc.On("Generate").Return("session-token", nil)
				m.tokenSvc.On("Hash", "session-token").Return("hashed-session-token")
				m.sessionSvc.On("Create", mock.Anything, m.uid, "hashed-session-token", mock.Anything, mock.Anything, mock.Anything).
					Return(&models.Session{ID: "sess-id", UserID: m.uid}, nil)
				m.userSvc.On("GetByID", mock.Anything, m.uid).Return(&models.User{ID: m.uid, Email: "user@example.com"}, nil)
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(&types.TOTPRecord{UserID: m.uid, Secret: "enc-secret", Enabled: true}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, reqCtx *models.RequestContext) {
				assert.Equal(t, true, reqCtx.Values[models.ContextAuthSuccess.String()])
				assert.Equal(t, "sess-id", reqCtx.Values[models.ContextSessionID.String()])
				assert.Equal(t, "session-token", reqCtx.Values[models.ContextSessionToken.String()])

				cleared := totptests.CookieFromRecorder(w, constants.CookieTOTPPending)
				require.NotNil(t, cleared)
				assert.Equal(t, -1, cleared.MaxAge)
				assert.Nil(t, totptests.CookieFromRecorder(w, constants.CookieTOTPTrusted))

				var resp types.VerifyTOTPResponse
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &resp))
				assert.Equal(t, "user-1", resp.User.ID)
			},
		},
		{
			name: "success_with_trusted_device",
			prepare: func(m *verifyTOTPFixture) {
				m.withCookie = true
				plainSecret := mustGenerateTOTPSecret(s.T())
				validCode := mustGenerateTOTPCode(s.T(), plainSecret)
				m.requestBody, _ = json.Marshal(types.VerifyTOTPRequest{Code: validCode, TrustDevice: true})

				verif := setupValidVerification(m.tokenSvc, m.verifSvc, m.pendingRaw, m.uid)
				m.verifSvc.On("Delete", mock.Anything, verif.ID).Return(nil)
				m.tokenSvc.On("Decrypt", "enc-secret").Return(plainSecret, nil)
				m.tokenSvc.On("Generate").Return("session-token", nil).Once()
				m.tokenSvc.On("Generate").Return("trusted-token", nil).Once()
				m.tokenSvc.On("Hash", "session-token").Return("hashed-session-token")
				m.tokenSvc.On("Hash", "trusted-token").Return("hashed-trusted-token")
				m.sessionSvc.On("Create", mock.Anything, m.uid, "hashed-session-token", mock.Anything, mock.Anything, mock.Anything).
					Return(&models.Session{ID: "sess-id", UserID: m.uid}, nil)
				m.userSvc.On("GetByID", mock.Anything, m.uid).Return(&models.User{ID: m.uid, Email: "user@example.com"}, nil)
				m.repo.On("GetByUserID", mock.Anything, m.uid).Return(&types.TOTPRecord{UserID: m.uid, Secret: "enc-secret", Enabled: true}, nil)
				m.repo.On("CreateTrustedDevice", mock.Anything, m.uid, "hashed-trusted-token", mock.Anything, mock.Anything).
					Return(&types.TrustedDevice{ID: "td-1"}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, _ *models.RequestContext) {
				trusted := totptests.CookieFromRecorder(w, constants.CookieTOTPTrusted)
				require.NotNil(t, trusted, "trusted device cookie should be set")
				assert.Equal(t, "trusted-token", trusted.Value)

				pending := totptests.CookieFromRecorder(w, constants.CookieTOTPPending)
				require.NotNil(t, pending)
				assert.Equal(t, -1, pending.MaxAge)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			t := s.T()
			m := newVerifyTOTPFixture()

			if tt.prepare != nil {
				tt.prepare(m)
			}

			h := &VerifyTOTPHandler{UseCase: buildVerifyTOTPUseCase(m.pluginCfg, m.tokenSvc, m.verifSvc, m.sessionSvc, m.userSvc, m.repo), PluginConfig: m.pluginCfg}
			req, w, reqCtx := m.newRequest(t)
			h.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w, reqCtx)
			}
		})
	}
}
