package bearer

import (
	"encoding/json"
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

func hookResponseMessage(t *testing.T, reqCtx *models.RequestContext) string {
	t.Helper()

	var payload struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &payload))
	return payload.Message
}

func TestValidateBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		header        string
		setupMock     func(*bearertests.MockJWTService)
		preSetActor   *models.Actor
		wantHandled   bool
		wantStatus    int
		wantMessage   string
		wantActor     *models.Actor
		wantSessionID string
	}{
		{
			name:        "actor_already_set",
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
			wantMessage: "missing authorization header",
		},
		{
			name:   "malformed_header",
			header: "Token abc",
			setupMock: func(m *bearertests.MockJWTService) {
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "invalid authorization format: expected 'Bearer <token>'",
		},
		{
			name:   "empty_token",
			header: "Bearer   ",
			setupMock: func(m *bearertests.MockJWTService) {
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "empty bearer token",
		},
		{
			name:   "invalid_token",
			header: "Bearer invalid-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "invalid-token").Return(nil, errors.New("invalid token")).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "Bearer token invalid or expired",
		},
		{
			name:   "valid_user_token",
			header: "Bearer valid-user-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "valid-user-token").Return(&models.Actor{ID: "user-1", Type: models.ActorUser}, nil).Once()
			},
			wantActor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{}, Claims: map[string]any{}},
		},
		{
			name:   "valid_machine_token",
			header: "Bearer valid-machine-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "valid-machine-token").Return(&models.Actor{ID: "client-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}, Scopes: []string{"read"}}, nil).Once()
			},
			wantActor: &models.Actor{ID: "client-1", Type: models.ActorMachine, Claims: map[string]any{"organization_id": "org-1"}, Scopes: []string{"read"}},
		},
		{
			name:   "valid_token_with_session_claim",
			header: "Bearer session-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "session-token").Return(&models.Actor{
					ID:     "user-1",
					Type:   models.ActorUser,
					Claims: map[string]any{models.ContextSessionID.String(): "session-1"},
				}, nil).Once()
			},
			wantActor: &models.Actor{
				ID:     "user-1",
				Type:   models.ActorUser,
				Scopes: []string{},
				Claims: map[string]any{models.ContextSessionID.String(): "session-1"},
			},
			wantSessionID: "session-1",
		},
		{
			name:   "valid_token_with_empty_session_claim",
			header: "Bearer empty-session-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "empty-session-token").Return(&models.Actor{
					ID:     "user-1",
					Type:   models.ActorUser,
					Claims: map[string]any{models.ContextSessionID.String(): ""},
				}, nil).Once()
			},
			wantActor: &models.Actor{
				ID:     "user-1",
				Type:   models.ActorUser,
				Scopes: []string{},
				Claims: map[string]any{models.ContextSessionID.String(): ""},
			},
		},
		{
			name:   "valid_token_with_non_string_session_claim",
			header: "Bearer numeric-session-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "numeric-session-token").Return(&models.Actor{
					ID:     "user-1",
					Type:   models.ActorUser,
					Claims: map[string]any{models.ContextSessionID.String(): 42},
				}, nil).Once()
			},
			wantActor: &models.Actor{
				ID:     "user-1",
				Type:   models.ActorUser,
				Scopes: []string{},
				Claims: map[string]any{models.ContextSessionID.String(): 42},
			},
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
			if tt.wantMessage != "" {
				require.Equal(t, tt.wantMessage, hookResponseMessage(t, reqCtx))
			}
			if tt.wantActor != nil {
				require.Equal(t, tt.wantActor, reqCtx.Actor)
			}
			if tt.wantSessionID != "" {
				require.Equal(t, tt.wantSessionID, reqCtx.Values[models.ContextSessionID.String()])
			} else {
				require.NotContains(t, reqCtx.Values, models.ContextSessionID.String())
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestValidateBearerTokenOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		header        string
		setupMock     func(*bearertests.MockJWTService)
		preSetActor   *models.Actor
		wantHandled   bool
		wantActor     *models.Actor
		wantSessionID string
	}{
		{
			name:        "actor_already_set_optional",
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
			name:   "malformed_header_optional",
			header: "Token abc",
			setupMock: func(m *bearertests.MockJWTService) {
			},
			wantHandled: false,
		},
		{
			name:   "empty_token_optional",
			header: "Bearer   ",
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
			wantActor: &models.Actor{ID: "user-1", Type: models.ActorUser, Scopes: []string{}, Claims: map[string]any{}},
		},
		{
			name:   "valid_token_with_session_claim_optional",
			header: "Bearer session-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "session-token").Return(&models.Actor{
					ID:     "user-1",
					Type:   models.ActorUser,
					Claims: map[string]any{models.ContextSessionID.String(): "session-1"},
				}, nil).Once()
			},
			wantActor: &models.Actor{
				ID:     "user-1",
				Type:   models.ActorUser,
				Scopes: []string{},
				Claims: map[string]any{models.ContextSessionID.String(): "session-1"},
			},
			wantSessionID: "session-1",
		},
		{
			name:   "valid_token_with_non_string_session_claim_optional",
			header: "Bearer numeric-session-token",
			setupMock: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "numeric-session-token").Return(&models.Actor{
					ID:     "user-1",
					Type:   models.ActorUser,
					Claims: map[string]any{models.ContextSessionID.String(): 42},
				}, nil).Once()
			},
			wantActor: &models.Actor{
				ID:     "user-1",
				Type:   models.ActorUser,
				Scopes: []string{},
				Claims: map[string]any{models.ContextSessionID.String(): 42},
			},
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
			if tt.wantSessionID != "" {
				require.Equal(t, tt.wantSessionID, reqCtx.Values[models.ContextSessionID.String()])
			} else {
				require.NotContains(t, reqCtx.Values, models.ContextSessionID.String())
			}
			mockSvc.AssertExpectations(t)
		})
	}
}
