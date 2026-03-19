package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

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

type GetTOTPURIHandlerSuite struct {
	suite.Suite
}

type getTOTPURIFixture struct {
	config   *types.TOTPPluginConfig
	userSvc  *internaltests.MockUserService
	tokenSvc *internaltests.MockTokenService
	totpSvc  *services.TOTPService
	repo     *totptests.MockTOTPRepo
}

type getTOTPURITestCase struct {
	name           string
	userID         *string
	prepare        func(m *getTOTPURIFixture)
	expectedStatus int
	checkResponse  func(t *testing.T, reqCtx *models.RequestContext)
}

func TestGetTOTPURIHandlerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GetTOTPURIHandlerSuite))
}

func (s *GetTOTPURIHandlerSuite) TestGetTOTPURIHandler_Table() {
	uid := "user-1"
	tests := []getTOTPURITestCase{
		{
			name:           "unauthenticated",
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "usecase_not_enabled",
			userID: internaltests.PtrString(uid),
			prepare: func(m *getTOTPURIFixture) {
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
			prepare: func(m *getTOTPURIFixture) {
				plainSecret, err := m.totpSvc.GenerateSecret()
				require.NoError(s.T(), err)

				m.repo.On("GetByUserID", mock.Anything, uid).Return(&types.TOTPRecord{
					UserID:  uid,
					Secret:  "enc-secret",
					Enabled: true,
				}, nil)
				m.userSvc.On("GetByID", mock.Anything, uid).Return(&models.User{ID: uid, Email: "user@example.com"}, nil)
				m.tokenSvc.On("Decrypt", "enc-secret").Return(plainSecret, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				var resp types.GetTOTPURIResponse
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &resp))
				assert.Contains(t, resp.TotpURI, "otpauth://totp/")
				assert.Contains(t, resp.TotpURI, "user@example.com")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			t := s.T()
			m := &getTOTPURIFixture{
				repo:     &totptests.MockTOTPRepo{},
				userSvc:  &internaltests.MockUserService{},
				tokenSvc: &internaltests.MockTokenService{},
				config:   &types.TOTPPluginConfig{},
				totpSvc:  services.NewTOTPService(6, 30),
			}

			if tt.prepare != nil {
				tt.prepare(m)
			}

			uc := usecases.NewGetTOTPURIUseCase(m.config, m.userSvc, m.tokenSvc, m.totpSvc, m.repo)
			h := &GetTOTPURIHandler{GlobalConfig: &models.Config{AppName: "MyApp"}, UseCase: uc}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, "/totp/get-uri", nil, tt.userID)
			h.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
		})
	}
}
