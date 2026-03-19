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
	totptests "github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/usecases"
)

type DisableHandlerSuite struct {
	suite.Suite
}

type disableHandlerFixture struct {
	repo     *totptests.MockTOTPRepo
	eventBus *internaltests.MockEventBus
	logger   *internaltests.MockLogger
}

type disableHandlerTestCase struct {
	name           string
	userID         *string
	prepare        func(m *disableHandlerFixture)
	expectedStatus int
	checkResponse  func(t *testing.T, reqCtx *models.RequestContext)
}

func TestDisableHandlerSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DisableHandlerSuite))
}

func (s *DisableHandlerSuite) TestDisableHandler_Table() {
	uid := "user-1"
	tests := []disableHandlerTestCase{
		{
			name:           "unauthenticated",
			userID:         nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "usecase_not_enabled",
			userID: internaltests.PtrString(uid),
			prepare: func(m *disableHandlerFixture) {
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
			prepare: func(m *disableHandlerFixture) {
				m.repo.On("GetByUserID", mock.Anything, uid).Return(&types.TOTPRecord{UserID: uid}, nil)
				m.repo.On("DeleteByUserID", mock.Anything, uid).Return(nil)
				m.repo.On("DeleteTrustedDevicesByUserID", mock.Anything, uid).Return(nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, reqCtx *models.RequestContext) {
				var resp map[string]string
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &resp))
				assert.Equal(t, "totp authentication disabled", resp["message"])
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			t := s.T()
			m := &disableHandlerFixture{
				repo:     &totptests.MockTOTPRepo{},
				eventBus: &internaltests.MockEventBus{},
				logger:   &internaltests.MockLogger{},
			}
			m.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			if tt.prepare != nil {
				tt.prepare(m)
			}

			uc := usecases.NewDisableUseCase(m.logger, m.eventBus, m.repo)
			h := &DisableHandler{UseCase: uc}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/totp/disable", nil, tt.userID)
			h.Handler().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			if tt.checkResponse != nil {
				tt.checkResponse(t, reqCtx)
			}
		})
	}
}
