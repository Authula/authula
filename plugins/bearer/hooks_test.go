package bearer

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	bearertests "github.com/Authula/authula/plugins/bearer/tests"
)

func newTestBearerPlugin(jwtSvc *bearertests.MockJWTService) *BearerPlugin {
	return &BearerPlugin{
		config:     BearerPluginConfig{HeaderName: "Authorization"},
		jwtService: jwtSvc,
	}
}

func newBearerRequestCtx(t *testing.T, header string) *models.RequestContext {
	t.Helper()
	req, _, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/test", nil, nil)
	if header != "" {
		req.Header.Set("Authorization", header)
		reqCtx.Headers = req.Header
	}
	return reqCtx
}

func TestValidateBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		header     string
		setupMock  func(*bearertests.MockJWTService)
		preSetActor *models.Actor
		wantHandled bool
		wantStatus  int
		wantActor   *models.Actor
	}{
		{
			name: "actor_already_set",
			preSetActor: &models.Actor{ID: "existing-user", Type: models.ActorUser},
			setupMock: func(m *bearertests.MockJWTService) {
			},
			wantHandled: false,
			wantActor:   &models.Actor{ID: "existing-user", Type: models.ActorUser},
		},
		{
			name:   "no_token",
			header: "",
			setupMock: func(m *bearertests.MockJWTService) {
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:   "invalid_token",
			header: "Bearer invalid-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "invalid-token").Return(nil, errors.New("invalid token")).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:   "valid_user_token",
			header: "Bearer valid-user-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "valid-user-token").Return(&models.Actor{ID: "user-1", Type: models.ActorUser}, nil).Once()
			},
			wantActor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{}, Metadata: map[string]any{}},
		},
		{
			name:   "valid_machine_token",
			header: "Bearer valid-machine-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "valid-machine-token").Return(&models.Actor{ID: "client-1", Type: models.ActorMachine, OrganizationID: internaltests.PtrString("org-1"), Scopes: []string{"read"}}, nil).Once()
			},
			wantActor: &models.Actor{ID: "client-1", Type: models.ActorMachine, OrganizationID: internaltests.PtrString("org-1"), Scopes: []string{"read"}, Metadata: map[string]any{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &bearertests.MockJWTService{}
			tt.setupMock(mockSvc)

			p := newTestBearerPlugin(mockSvc)
			reqCtx := newBearerRequestCtx(t, tt.header)
			if tt.preSetActor != nil {
				reqCtx.Actor = tt.preSetActor
			}

			err := p.validateBearerToken(reqCtx)
			require.NoError(t, err)

			require.Equal(t, tt.wantHandled, reqCtx.Handled)
			if tt.wantStatus != 0 {
				require.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			}
			if tt.wantActor != nil {
				require.Equal(t, tt.wantActor, reqCtx.Actor)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestValidateBearerTokenOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		header      string
		setupMock   func(*bearertests.MockJWTService)
		preSetActor *models.Actor
		wantHandled bool
		wantActor   *models.Actor
	}{
		{
			name: "actor_already_set_optional",
			preSetActor: &models.Actor{ID: "existing-user", Type: models.ActorUser},
			setupMock: func(m *bearertests.MockJWTService) {
			},
			wantHandled: false,
			wantActor:   &models.Actor{ID: "existing-user", Type: models.ActorUser},
		},
		{
			name:   "no_token_optional",
			header: "",
			setupMock: func(m *bearertests.MockJWTService) {
			},
			wantHandled: false,
		},
		{
			name:   "invalid_token_optional",
			header: "Bearer invalid-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "invalid-token").Return(nil, errors.New("invalid token")).Once()
			},
			wantHandled: false,
		},
		{
			name:   "valid_token_optional",
			header: "Bearer valid-user-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "valid-user-token").Return(&models.Actor{ID: "user-1", Type: models.ActorUser}, nil).Once()
			},
			wantActor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{}, Metadata: map[string]any{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &bearertests.MockJWTService{}
			tt.setupMock(mockSvc)

			p := newTestBearerPlugin(mockSvc)
			reqCtx := newBearerRequestCtx(t, tt.header)
			if tt.preSetActor != nil {
				reqCtx.Actor = tt.preSetActor
			}

			err := p.validateBearerTokenOptional(reqCtx)
			require.NoError(t, err)

			require.Equal(t, tt.wantHandled, reqCtx.Handled)
			if tt.wantActor != nil {
				require.Equal(t, tt.wantActor, reqCtx.Actor)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}
