package admin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	adminservices "github.com/Authula/authula/plugins/admin/services"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
)

type hookTestPlugin struct {
	plugin           *AdminPlugin
	userStateRepo    *admintests.MockUserStateRepository
	sessionStateRepo *admintests.MockSessionStateRepository
	sessionSvc       *internaltests.MockSessionService
	tokenSvc         *internaltests.MockTokenService
}

func newHookTestPlugin(t *testing.T) *hookTestPlugin {
	t.Helper()

	userStateRepo := &admintests.MockUserStateRepository{}
	sessionStateRepo := &admintests.MockSessionStateRepository{}
	impersonationRepo := &admintests.MockImpersonationRepository{}
	stateService := adminservices.NewStateService(userStateRepo, sessionStateRepo, impersonationRepo)

	plugin := New(types.AdminPluginConfig{})
	plugin.globalConfig = &models.Config{Session: models.SessionConfig{CookieName: "authula.session_token"}}
	plugin.pluginCtx = &models.PluginContext{GetConfig: func() *models.Config { return plugin.globalConfig }}
	plugin.Api = NewAPI(nil, nil, stateService, nil, nil, userStateRepo, sessionStateRepo)

	return &hookTestPlugin{
		plugin:           plugin,
		userStateRepo:    userStateRepo,
		sessionStateRepo: sessionStateRepo,
		sessionSvc:       &internaltests.MockSessionService{},
		tokenSvc:         &internaltests.MockTokenService{},
	}
}

func newHookRequestContext(sessionValue any, cookie *http.Cookie) (*models.RequestContext, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	rc := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values:         map[string]any{},
		Actor:          &models.Actor{ID: "user-1", Type: models.ActorUser},
	}
	if sessionValue != nil {
		rc.Values[models.ContextSessionID.String()] = sessionValue
	}
	return rc, w
}

func TestAdminPlugin_EnforceState(t *testing.T) {
	t.Parallel()

	future := time.Now().UTC().Add(time.Hour)
	past := time.Now().UTC().Add(-time.Hour)
	lookupErr := errors.New("state lookup failure")
	sessionLookupErr := errors.New("session lookup failure")

	tests := []struct {
		name         string
		actor        *models.Actor
		sessionValue any
		setup        func(tt *hookTestPlugin)
		wantHandled  bool
		wantStatus   int
		wantMessage  string
	}{
		{
			name:  "nil actor is ignored",
			actor: nil,
		},
		{
			name:  "non user actor is ignored",
			actor: &models.Actor{ID: "svc", Type: models.ActorMachine},
		},
		{
			name:  "empty actor id is ignored",
			actor: &models.Actor{ID: "", Type: models.ActorUser},
		},
		{
			name:  "state lookup error returns internal error",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return((*types.AdminUserState)(nil), lookupErr).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusInternalServerError,
			wantMessage: "failed to evaluate user state",
		},
		{
			name:  "banned without expiry is forbidden",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: true}, nil).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusForbidden,
			wantMessage: "user is banned",
		},
		{
			name:  "banned until future is forbidden",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: true, BannedUntil: &future}, nil).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusForbidden,
			wantMessage: "user is banned",
		},
		{
			name:  "expired ban is allowed",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1", Banned: true, BannedUntil: &past}, nil).Once()
			},
		},
		{
			name:  "unbanned without session id passes through",
			actor: &models.Actor{ID: "user-1", Type: models.ActorUser},
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
			},
		},
		{
			name:         "non string session id passes through",
			actor:        &models.Actor{ID: "user-1", Type: models.ActorUser},
			sessionValue: 42,
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
			},
		},
		{
			name:         "empty session id passes through",
			actor:        &models.Actor{ID: "user-1", Type: models.ActorUser},
			sessionValue: "",
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
			},
		},
		{
			name:         "session state lookup error returns internal error",
			actor:        &models.Actor{ID: "user-1", Type: models.ActorUser},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return((*types.AdminSessionState)(nil), sessionLookupErr).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusInternalServerError,
			wantMessage: "failed to evaluate session state",
		},
		{
			name:         "revoked session is unauthorized",
			actor:        &models.Actor{ID: "user-1", Type: models.ActorUser},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&types.AdminSessionState{SessionID: "sess-1", RevokedAt: &past}, nil).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "session is revoked",
		},
		{
			name:         "active session passes through",
			actor:        &models.Actor{ID: "user-1", Type: models.ActorUser},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.userStateRepo.On("GetByUserID", mock.Anything, "user-1").Return(&types.AdminUserState{UserID: "user-1"}, nil).Once()
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&types.AdminSessionState{SessionID: "sess-1"}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ttPlugin := newHookTestPlugin(t)
			if tt.setup != nil {
				tt.setup(ttPlugin)
			}

			rc, _ := newHookRequestContext(tt.sessionValue, nil)
			rc.Actor = tt.actor

			err := ttPlugin.plugin.enforceState(rc)
			require.NoError(t, err)

			assert.Equal(t, tt.wantHandled, rc.Handled)
			if tt.wantHandled {
				internaltests.AssertErrorMessage(t, rc, tt.wantStatus, tt.wantMessage)
			} else {
				assert.Zero(t, rc.ResponseStatus)
			}

			ttPlugin.userStateRepo.AssertExpectations(t)
			ttPlugin.sessionStateRepo.AssertExpectations(t)
		})
	}
}

func TestAdminPlugin_ResolveImpersonationOriginalSession(t *testing.T) {
	t.Parallel()

	hashedCookie := "hashed-original"
	expired := time.Now().UTC().Add(-time.Hour)
	valid := time.Now().UTC().Add(time.Hour)

	tests := []struct {
		name              string
		actor             *models.Actor
		cookie            *http.Cookie
		sessionValue      any
		setup             func(tt *hookTestPlugin)
		wantCookieCleared bool
		wantImpersonator  string
		wantNoClaims      bool
	}{
		{
			name:         "nil actor is ignored",
			actor:        nil,
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
		},
		{
			name:         "non user actor is ignored",
			actor:        &models.Actor{ID: "svc", Type: models.ActorMachine},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
		},
		{
			name:         "existing claims are left untouched",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor"}},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
			wantNoClaims: false,
		},
		{
			name:         "missing cookie is ignored",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			sessionValue: "sess-1",
			wantNoClaims: true,
		},
		{
			name:         "missing session id is ignored",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			wantNoClaims: true,
		},
		{
			name:         "non string session id is ignored",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: 42,
			wantNoClaims: true,
		},
		{
			name:         "empty session id is ignored",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "",
			wantNoClaims: true,
		},
		{
			name:         "session state lookup error is ignored",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return((*types.AdminSessionState)(nil), errors.New("lookup failure")).Once()
			},
			wantNoClaims: true,
		},
		{
			name:         "state without impersonator is ignored",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&types.AdminSessionState{SessionID: "sess-1"}, nil).Once()
			},
			wantNoClaims: true,
		},
		{
			name:         "unresolvable original session clears cookie",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&types.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor")}, nil).Once()
				tt.tokenSvc.On("Hash", "orig").Return(hashedCookie).Once()
				tt.sessionSvc.On("GetByToken", mock.Anything, hashedCookie).Return((*models.Session)(nil), errors.New("not found")).Once()
			},
			wantCookieCleared: true,
			wantNoClaims:      true,
		},
		{
			name:         "mismatched impersonator session clears cookie",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&types.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor"), ImpersonatorSessionID: new("other-sess")}, nil).Once()
				tt.tokenSvc.On("Hash", "orig").Return(hashedCookie).Once()
				tt.sessionSvc.On("GetByToken", mock.Anything, hashedCookie).Return(&models.Session{ID: "orig-sess", UserID: "actor", ExpiresAt: valid}, nil).Once()
			},
			wantCookieCleared: true,
			wantNoClaims:      true,
		},
		{
			name:         "expired original session clears cookie",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&types.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor"), ImpersonatorSessionID: new("orig-sess")}, nil).Once()
				tt.tokenSvc.On("Hash", "orig").Return(hashedCookie).Once()
				tt.sessionSvc.On("GetByToken", mock.Anything, hashedCookie).Return(&models.Session{ID: "orig-sess", UserID: "actor", ExpiresAt: expired}, nil).Once()
			},
			wantCookieCleared: true,
			wantNoClaims:      true,
		},
		{
			name:         "valid original session restores impersonator claims",
			actor:        &models.Actor{ID: "target", Type: models.ActorUser},
			cookie:       &http.Cookie{Name: "authula.session_token.original", Value: "orig"},
			sessionValue: "sess-1",
			setup: func(tt *hookTestPlugin) {
				tt.sessionStateRepo.On("GetBySessionID", mock.Anything, "sess-1").Return(&types.AdminSessionState{SessionID: "sess-1", ImpersonatorUserID: new("actor"), ImpersonatorSessionID: new("orig-sess")}, nil).Once()
				tt.tokenSvc.On("Hash", "orig").Return(hashedCookie).Once()
				tt.sessionSvc.On("GetByToken", mock.Anything, hashedCookie).Return(&models.Session{ID: "orig-sess", UserID: "actor", ExpiresAt: valid}, nil).Once()
			},
			wantImpersonator: "actor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ttPlugin := newHookTestPlugin(t)
			ttPlugin.plugin.tokenService = ttPlugin.tokenSvc
			ttPlugin.plugin.sessionService = ttPlugin.sessionSvc
			if tt.setup != nil {
				tt.setup(ttPlugin)
			}

			rc, w := newHookRequestContext(tt.sessionValue, tt.cookie)
			rc.Actor = tt.actor

			err := ttPlugin.plugin.resolveImpersonationOriginalSession(rc)
			require.NoError(t, err)

			if tt.wantImpersonator != "" {
				require.NotNil(t, rc.Actor)
				got, ok := rc.Actor.Claims[adminconstants.ImpersonatorID]
				require.True(t, ok)
				assert.Equal(t, tt.wantImpersonator, got)
				assert.Equal(t, "orig", rc.Actor.Claims[adminconstants.ImpersonatorOriginalSessionToken])
			}
			if tt.wantNoClaims {
				if rc.Actor != nil {
					assert.NotContains(t, rc.Actor.Claims, adminconstants.ImpersonatorID)
				}
			}

			cleared := false
			for _, c := range w.Result().Cookies() {
				if c.Name == "authula.session_token"+adminconstants.OriginalSessionCookieSuffix && c.MaxAge == -1 {
					cleared = true
				}
			}
			assert.Equal(t, tt.wantCookieCleared, cleared)

			ttPlugin.sessionStateRepo.AssertExpectations(t)
			ttPlugin.tokenSvc.AssertExpectations(t)
			ttPlugin.sessionSvc.AssertExpectations(t)
		})
	}
}

func TestAdminPlugin_AddImpersonationWhitelistScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		actor      *models.Actor
		wantScopes []string
	}{
		{
			name:  "nil actor is ignored",
			actor: nil,
		},
		{
			name:  "non user actor is ignored",
			actor: &models.Actor{ID: "svc", Type: models.ActorMachine},
		},
		{
			name:       "user without impersonator claim is ignored",
			actor:      &models.Actor{ID: "user", Type: models.ActorUser, Scopes: []string{"existing"}},
			wantScopes: []string{"existing"},
		},
		{
			name:  "empty impersonator claim is ignored",
			actor: &models.Actor{ID: "user", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: ""}},
		},
		{
			name:  "adds missing whitelist scopes",
			actor: &models.Actor{ID: "user", Type: models.ActorUser, Claims: map[string]any{adminconstants.ImpersonatorID: "actor"}},
			wantScopes: []string{
				adminconstants.SessionStateReadPermission,
				adminconstants.ImpersonationsStopPermission,
			},
		},
		{
			name: "does not duplicate existing scopes",
			actor: &models.Actor{
				ID:     "user",
				Type:   models.ActorUser,
				Claims: map[string]any{adminconstants.ImpersonatorID: "actor"},
				Scopes: []string{adminconstants.SessionStateReadPermission, "other"},
			},
			wantScopes: []string{
				adminconstants.SessionStateReadPermission,
				"other",
				adminconstants.ImpersonationsStopPermission,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(types.AdminPluginConfig{})
			rc, _ := newHookRequestContext(nil, nil)
			rc.Actor = tt.actor

			require.NoError(t, plugin.addImpersonationWhitelistScopes(rc))

			if tt.wantScopes != nil {
				require.NotNil(t, rc.Actor)
				assert.Equal(t, tt.wantScopes, rc.Actor.Scopes)
			}
		})
	}
}

func TestAdminPlugin_BuildRoutes(t *testing.T) {
	t.Parallel()

	plugin := New(types.AdminPluginConfig{})
	plugin.globalConfig = &models.Config{}
	plugin.useCases = testAdminUseCases()

	routes := plugin.Routes()
	require.Len(t, routes, 28)

	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		assert.NotEmpty(t, route.Method)
		assert.NotEmpty(t, route.Path)
		assert.NotNil(t, route.Handler)
		assert.NotEmpty(t, route.Middleware)

		key := route.Method + " " + route.Path
		_, duplicate := seen[key]
		assert.False(t, duplicate, "duplicate route %s", key)
		seen[key] = struct{}{}
	}

	ruc := newRouteUseCases(plugin)
	assert.NotEqual(t, routeUseCases{}, ruc)
}
