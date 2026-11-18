package bearer

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	bearertests "github.com/Authula/authula/plugins/bearer/tests"
)

func TestBearerPlugin_Metadata(t *testing.T) {
	t.Parallel()

	plugin := New(BearerPluginConfig{})
	metadata := plugin.Metadata()

	require.NotEmpty(t, metadata.ID)
	require.NotEmpty(t, metadata.Version)
	require.NotEmpty(t, metadata.Description)
}

func TestBearerPlugin_Config(t *testing.T) {
	t.Parallel()

	cfg := BearerPluginConfig{HeaderName: "Custom-Auth", Enabled: true}
	plugin := New(cfg)

	returnedCfg := plugin.Config()
	require.Equal(t, cfg, returnedCfg)
}

func TestBearerPlugin_Init(t *testing.T) {
	t.Parallel()

	t.Run("missing_jwt_service", func(t *testing.T) {
		t.Parallel()
		reg := &internaltests.MockServiceRegistry{}
		reg.On("Get", models.ServiceJWT.String()).Return(nil).Once()

		plugin := New(BearerPluginConfig{})
		err := plugin.Init(&models.PluginContext{
			Logger:          &internaltests.MockLogger{},
			ServiceRegistry: reg,
			GetConfig:       func() *models.Config { return &models.Config{} },
		})
		require.Error(t, err)
		reg.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc := &bearertests.MockJWTService{}
		reg := &internaltests.MockServiceRegistry{}
		reg.On("Get", models.ServiceJWT.String()).Return(mockSvc).Once()

		plugin := New(BearerPluginConfig{})
		err := plugin.Init(&models.PluginContext{
			Logger:          &internaltests.MockLogger{},
			ServiceRegistry: reg,
			GetConfig:       func() *models.Config { return &models.Config{} },
		})
		require.NoError(t, err)
		require.Equal(t, mockSvc, plugin.jwtService)
		reg.AssertExpectations(t)
	})

	t.Run("invalid_plugin_config", func(t *testing.T) {
		t.Parallel()
		reg := &internaltests.MockServiceRegistry{}

		plugin := New(BearerPluginConfig{})
		err := plugin.Init(&models.PluginContext{
			Logger:          &internaltests.MockLogger{},
			ServiceRegistry: reg,
			GetConfig: func() *models.Config {
				return &models.Config{
					Plugins: models.PluginsConfig{
						models.PluginBearer.String(): map[string]any{"enabled": "not-a-bool"},
					},
				}
			},
		})
		require.Error(t, err)
		reg.AssertNotCalled(t, "Get", mock.Anything)
		reg.AssertExpectations(t)
	})

	t.Run("preparsed_config_is_applied", func(t *testing.T) {
		t.Parallel()
		mockSvc := &bearertests.MockJWTService{}
		reg := &internaltests.MockServiceRegistry{}
		reg.On("Get", models.ServiceJWT.String()).Return(mockSvc).Once()

		plugin := New(BearerPluginConfig{})
		err := plugin.Init(&models.PluginContext{
			Logger:          &internaltests.MockLogger{},
			ServiceRegistry: reg,
			GetConfig: func() *models.Config {
				return &models.Config{
					Plugins: models.PluginsConfig{models.PluginBearer.String(): map[string]any{}},
					PreParsedConfigs: map[string]any{
						models.PluginBearer.String(): BearerPluginConfig{Enabled: true, HeaderName: "X-Token"},
					},
				}
			},
		})
		require.NoError(t, err)
		require.True(t, plugin.config.Enabled)
		require.Equal(t, "X-Token", plugin.config.HeaderName)
		reg.AssertExpectations(t)
	})

	t.Run("plugin_map_config_is_parsed", func(t *testing.T) {
		t.Parallel()
		mockSvc := &bearertests.MockJWTService{}
		reg := &internaltests.MockServiceRegistry{}
		reg.On("Get", models.ServiceJWT.String()).Return(mockSvc).Once()

		plugin := New(BearerPluginConfig{})
		err := plugin.Init(&models.PluginContext{
			Logger:          &internaltests.MockLogger{},
			ServiceRegistry: reg,
			GetConfig: func() *models.Config {
				return &models.Config{
					Plugins: models.PluginsConfig{
						models.PluginBearer.String(): map[string]any{
							"enabled":     true,
							"header_name": "X-Auth-Token",
						},
					},
				}
			},
		})
		require.NoError(t, err)
		require.True(t, plugin.config.Enabled)
		require.Equal(t, "X-Auth-Token", plugin.config.HeaderName)
		reg.AssertExpectations(t)
	})
}

func TestBearerPlugin_Close(t *testing.T) {
	t.Parallel()

	plugin := newTestBearerPlugin(&bearertests.MockJWTService{})
	require.NoError(t, plugin.Close())
}

func TestBearerPlugin_Hooks(t *testing.T) {
	t.Parallel()

	plugin := newTestBearerPlugin(&bearertests.MockJWTService{})
	hooks := plugin.Hooks()

	require.Len(t, hooks, 2)

	require.Equal(t, models.HookBefore, hooks[0].Stage)
	require.Equal(t, HookIDBearerAuth.String(), hooks[0].PluginID)
	require.Equal(t, 10, hooks[0].Order)
	require.NotNil(t, hooks[0].Handler)

	require.Equal(t, models.HookBefore, hooks[1].Stage)
	require.Equal(t, HookIDBearerAuthOptional.String(), hooks[1].PluginID)
	require.Equal(t, 10, hooks[1].Order)
	require.NotNil(t, hooks[1].Handler)
}

func TestBearerHookID_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   BearerHookID
		want string
	}{
		{name: "required auth hook", id: HookIDBearerAuth, want: "bearer.auth"},
		{name: "optional auth hook", id: HookIDBearerAuthOptional, want: "bearer.auth.optional"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.id.String())
		})
	}
}

type middlewareOutcome struct {
	called  bool
	actor   *models.Actor
	userID  string
	userSet bool
}

func serveBearerMiddleware(t *testing.T, plugin *BearerPlugin, optional bool, headerName, headerValue string) (*httptest.ResponseRecorder, middlewareOutcome) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if headerValue != "" {
		req.Header.Set(headerName, headerValue)
	}

	outcome := middlewareOutcome{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outcome.called = true
		outcome.actor, _ = models.GetActorFromRequest(r)
		outcome.userID, outcome.userSet = r.Context().Value(models.ContextUserID).(string)
		w.WriteHeader(http.StatusOK)
	})

	handler := plugin.AuthMiddleware()
	if optional {
		handler = plugin.OptionalAuthMiddleware()
	}

	rec := httptest.NewRecorder()
	handler(next).ServeHTTP(rec, req)

	return rec, outcome
}

func responseMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var payload struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload.Message
}

func TestBearerPlugin_AuthMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		headerName  string
		header      string
		jwtSetup    func(*bearertests.MockJWTService)
		wantStatus  int
		wantMessage string
		wantCalled  bool
		wantActorID string
		wantUserID  string
	}{
		{
			name:        "missing_header",
			header:      "",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "missing authorization header",
			wantCalled:  false,
		},
		{
			name:        "wrong_scheme",
			header:      "Basic abc",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "invalid authorization format: expected 'Bearer <token>'",
			wantCalled:  false,
		},
		{
			name:        "scheme_without_token",
			header:      "Bearer",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "invalid authorization format: expected 'Bearer <token>'",
			wantCalled:  false,
		},
		{
			name:        "empty_token",
			header:      "Bearer   ",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "empty bearer token",
			wantCalled:  false,
		},
		{
			name:   "invalid_token",
			header: "Bearer bad-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "bad-token").Return(nil, errors.New("token expired")).Once()
			},
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "token expired",
			wantCalled:  false,
		},
		{
			name:   "lowercase_scheme",
			header: "bearer lower-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "lower-token").Return(&models.Actor{ID: "user-1", Type: models.ActorUser}, nil).Once()
			},
			wantCalled:  true,
			wantActorID: "user-1",
			wantUserID:  "user-1",
		},
		{
			name:   "extra_whitespace",
			header: "Bearer     spaced-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "spaced-token").Return(&models.Actor{ID: "user-2", Type: models.ActorUser}, nil).Once()
			},
			wantCalled:  true,
			wantActorID: "user-2",
			wantUserID:  "user-2",
		},
		{
			name:       "custom_header_name",
			headerName: "X-API-Token",
			header:     "Bearer custom-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "custom-token").Return(&models.Actor{ID: "user-3", Type: models.ActorUser}, nil).Once()
			},
			wantCalled:  true,
			wantActorID: "user-3",
			wantUserID:  "user-3",
		},
		{
			name:   "actor_without_id_does_not_set_user_id",
			header: "Bearer anonymous-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "anonymous-token").Return(&models.Actor{Type: models.ActorMachine}, nil).Once()
			},
			wantCalled:  true,
			wantActorID: "",
			wantUserID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &bearertests.MockJWTService{}
			if tt.jwtSetup != nil {
				tt.jwtSetup(mockSvc)
			}

			plugin := newTestBearerPlugin(mockSvc)
			headerName := "Authorization"
			if tt.headerName != "" {
				headerName = tt.headerName
				plugin.config.HeaderName = tt.headerName
			}

			rec, outcome := serveBearerMiddleware(t, plugin, false, headerName, tt.header)

			require.Equal(t, tt.wantCalled, outcome.called)
			if tt.wantCalled {
				require.Equal(t, http.StatusOK, rec.Code)
				require.NotNil(t, outcome.actor)
				require.Equal(t, tt.wantActorID, outcome.actor.ID)
				require.Equal(t, tt.wantUserID, outcome.userID)
				require.Equal(t, tt.wantUserID != "", outcome.userSet)
			} else {
				require.Equal(t, tt.wantStatus, rec.Code)
				require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
				require.Equal(t, tt.wantMessage, responseMessage(t, rec))
				require.Nil(t, outcome.actor)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestBearerPlugin_OptionalAuthMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		header      string
		jwtSetup    func(*bearertests.MockJWTService)
		wantStatus  int
		wantActorID string
		wantUserID  string
		wantActor   bool
	}{
		{
			name:       "missing_header_allows_anonymous",
			header:     "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong_scheme_allows_anonymous",
			header:     "Basic abc",
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty_token_allows_anonymous",
			header:     "Bearer ",
			wantStatus: http.StatusOK,
		},
		{
			name:   "invalid_token_allows_anonymous",
			header: "Bearer bad-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "bad-token").Return(nil, errors.New("token expired")).Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "valid_token_sets_actor",
			header: "bearer valid-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "valid-token").Return(&models.Actor{ID: "user-1", Type: models.ActorUser}, nil).Once()
			},
			wantStatus:  http.StatusOK,
			wantActor:   true,
			wantActorID: "user-1",
			wantUserID:  "user-1",
		},
		{
			name:   "actor_without_id_does_not_set_user_id",
			header: "Bearer anonymous-token",
			jwtSetup: func(m *bearertests.MockJWTService) {
				m.On("ValidateToken", mock.Anything, "anonymous-token").Return(&models.Actor{Type: models.ActorMachine}, nil).Once()
			},
			wantStatus: http.StatusOK,
			wantActor:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := &bearertests.MockJWTService{}
			if tt.jwtSetup != nil {
				tt.jwtSetup(mockSvc)
			}

			plugin := newTestBearerPlugin(mockSvc)
			rec, outcome := serveBearerMiddleware(t, plugin, true, "Authorization", tt.header)

			require.True(t, outcome.called)
			require.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantActor {
				require.NotNil(t, outcome.actor)
				require.Equal(t, tt.wantActorID, outcome.actor.ID)
			} else {
				require.Nil(t, outcome.actor)
			}
			require.Equal(t, tt.wantUserID, outcome.userID)
			require.Equal(t, tt.wantUserID != "", outcome.userSet)

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestBearerPlugin_writeUnauthorized(t *testing.T) {
	t.Parallel()

	plugin := newTestBearerPlugin(&bearertests.MockJWTService{})
	rec := httptest.NewRecorder()

	plugin.writeUnauthorized(rec, errors.New("missing authorization header"))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.JSONEq(t, `{"message":"missing authorization header"}`, rec.Body.String())
}
