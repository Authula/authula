package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/handlers"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
)

type authorizeUseCaseStub struct {
	result *usecases.AuthorizeResult
	err    error
	req    *types.AuthorizeRequest
}

func (s *authorizeUseCaseStub) Authorize(_ context.Context, req *types.AuthorizeRequest) (*usecases.AuthorizeResult, error) {
	s.req = req
	return s.result, s.err
}

type callbackUseCaseStub struct {
	result *types.CallbackResult
	err    error
	req    *types.CallbackRequest
}

func (s *callbackUseCaseStub) Callback(_ context.Context, req *types.CallbackRequest, _ *string, _ *string) (*types.CallbackResult, error) {
	s.req = req
	return s.result, s.err
}

func newHandlerRequest(t *testing.T, method, target string) (*http.Request, *models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	recorder := httptest.NewRecorder()
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: recorder,
		Values:         make(map[string]any),
	}
	req = req.WithContext(models.NewContextWithRequestContext(req.Context(), reqCtx))
	reqCtx.Request = req
	return req, reqCtx, recorder
}

func findCookie(t *testing.T, recorder *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestAuthorizeHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		result           *usecases.AuthorizeResult
		err              error
		cookie           handlers.CookieOptions
		actor            *models.Actor
		provider         string
		wantStatus       int
		wantActorID      string
		wantRedirectTo   string
		wantBodyContains string
		checkCookies     func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success sets flow cookies from config",
			result: &usecases.AuthorizeResult{
				AuthorizationURL: "https://provider.example/authorize?state=1",
				StateCookie:      "signed-state",
				RedirectCookie:   "signed-redirect",
				VerifierCookie:   new("signed-verifier"),
			},
			cookie: handlers.CookieOptions{
				Secure:   false,
				Domain:   "api.example.com",
				SameSite: http.SameSiteStrictMode,
				MaxAge:   5 * time.Minute,
			},
			actor:          &models.Actor{ID: "user-1", Type: models.ActorUser},
			provider:       "google",
			wantStatus:     http.StatusOK,
			wantActorID:    "user-1",
			wantRedirectTo: "/welcome",
			checkCookies: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				stateCookie := findCookie(t, recorder, constants.CookieState)
				require.NotNil(t, stateCookie)
				assert.Equal(t, "signed-state", stateCookie.Value)
				assert.Equal(t, "api.example.com", stateCookie.Domain)
				assert.False(t, stateCookie.Secure, "Secure must follow config, not X-Forwarded-Proto")
				assert.Equal(t, http.SameSiteLaxMode, stateCookie.SameSite, "OAuth flow cookies must be Lax even when the session is strict")

				verifierCookie := findCookie(t, recorder, constants.CookieVerifier)
				require.NotNil(t, verifierCookie)
				assert.Equal(t, "signed-verifier", verifierCookie.Value)
			},
		},
		{
			name:             "error maps without leaking details",
			err:              constants.ErrProviderNotFound,
			provider:         "missing",
			wantStatus:       http.StatusNotFound,
			wantBodyContains: "provider not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &authorizeUseCaseStub{result: tt.result, err: tt.err}
			handler := &handlers.AuthorizeHandler{
				UseCase: stub,
				Cookie:  tt.cookie,
				Logger:  &internaltests.MockLogger{},
			}

			target := "/oauth2/authorize/" + tt.provider
			if tt.wantRedirectTo != "" {
				target += "?redirect_to=/welcome"
			}
			req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, target)
			req.SetPathValue("provider", tt.provider)
			req.Header.Set("X-Forwarded-Proto", "https")
			reqCtx.Actor = tt.actor

			handler.Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
			if tt.wantBodyContains != "" {
				assert.Contains(t, string(reqCtx.ResponseBody), tt.wantBodyContains)
			}
			if tt.actor != nil {
				assert.Equal(t, tt.wantActorID, stub.req.ActorID)
				assert.Equal(t, tt.wantRedirectTo, stub.req.RedirectTo)
			}
			if tt.checkCookies != nil {
				tt.checkCookies(t, recorder)
			}
		})
	}
}

func TestCallbackHandler_ClearsFlowCookiesOnEveryPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cookie     *http.Cookie
		stub       *callbackUseCaseStub
		wantStatus int
	}{
		{
			name:       "missing state cookie",
			stub:       &callbackUseCaseStub{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "usecase error",
			cookie:     &http.Cookie{Name: constants.CookieState, Value: "signed-state"},
			stub:       &callbackUseCaseStub{err: constants.ErrProviderFailure},
			wantStatus: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := &handlers.CallbackHandler{UseCase: tt.stub, Logger: &internaltests.MockLogger{}}
			req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/callback/google?code=1&state=1")
			req.SetPathValue("provider", "google")
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
				req.AddCookie(&http.Cookie{Name: constants.CookieRedirectTo, Value: "signed-redirect"})
				req.AddCookie(&http.Cookie{Name: constants.CookieVerifier, Value: "signed-verifier"})
			}

			handler.Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			for _, name := range []string{constants.CookieState, constants.CookieRedirectTo, constants.CookieVerifier} {
				cookie := findCookie(t, recorder, name)
				require.NotNil(t, cookie, "expected a clearing cookie for %s", name)
				assert.Less(t, cookie.MaxAge, 0)
			}
		})
	}
}

func TestCallbackHandler(t *testing.T) {
	t.Parallel()

	safeSession := func() *models.Session {
		return &models.Session{
			ID:        "session-1",
			UserID:    "user-1",
			Token:     "hashed-token",
			ExpiresAt: time.Now().Add(time.Hour),
			IPAddress: new("203.0.113.9"),
			UserAgent: new("secret-agent"),
		}
	}

	tests := []struct {
		name             string
		cookie           handlers.CookieOptions
		result           func() *types.CallbackResult
		wantStatus       int
		wantRedirectURL  string
		wantSessionToken string
		wantSafeBody     bool
		wantNoToken      bool
		wantCacheControl string
	}{
		{
			name:   "success returns a safe response and sets the session",
			cookie: handlers.CookieOptions{Secure: true},
			result: func() *types.CallbackResult {
				return &types.CallbackResult{
					User:         &models.User{ID: "user-1", Email: "user@example.com"},
					Session:      safeSession(),
					SessionToken: "plain-token",
				}
			},
			wantStatus:       http.StatusOK,
			wantSessionToken: "plain-token",
			wantSafeBody:     true,
			wantCacheControl: "no-store",
		},
		{
			name: "reused session does not mint a new token",
			result: func() *types.CallbackResult {
				return &types.CallbackResult{
					User:          &models.User{ID: "user-1"},
					Session:       safeSession(),
					ReusedSession: true,
				}
			},
			wantStatus:       http.StatusOK,
			wantNoToken:      true,
			wantCacheControl: "no-store",
		},
		{
			name: "redirect is applied only after usecase success",
			result: func() *types.CallbackResult {
				return &types.CallbackResult{
					User:       &models.User{ID: "user-1"},
					Session:    safeSession(),
					RedirectTo: "/welcome",
				}
			},
			wantStatus:      http.StatusFound,
			wantRedirectURL: "/welcome",
			wantNoToken:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &callbackUseCaseStub{result: tt.result()}
			handler := &handlers.CallbackHandler{
				UseCase: stub,
				Cookie:  tt.cookie,
				Logger:  &internaltests.MockLogger{},
			}

			req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/callback/google?code=1&state=1")
			req.SetPathValue("provider", "google")
			req.AddCookie(&http.Cookie{Name: constants.CookieState, Value: "signed-state"})

			handler.Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.Equal(t, tt.wantCacheControl, reqCtx.ResponseHeaders.Get("Cache-Control"))
			assert.Equal(t, "session-1", reqCtx.Values[models.ContextSessionID.String()])
			assert.Equal(t, tt.wantRedirectURL, reqCtx.RedirectURL)

			if tt.wantSessionToken != "" {
				assert.Equal(t, tt.wantSessionToken, reqCtx.Values[models.ContextSessionToken.String()])
			}
			if tt.wantNoToken {
				assert.NotContains(t, reqCtx.Values, models.ContextSessionToken.String())
			}

			if tt.wantSafeBody {
				assert.Equal(t, "user-1", reqCtx.Actor.ID)

				var body map[string]any
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &body))
				sessionBody := body["session"].(map[string]any)
				assert.NotContains(t, sessionBody, "token")
				assert.NotContains(t, sessionBody, "ip_address")
				assert.NotContains(t, sessionBody, "user_agent")
				assert.NotContains(t, string(reqCtx.ResponseBody), "plain-token")
				assert.NotContains(t, string(reqCtx.ResponseBody), "hashed-token")
			}
		})
	}
}

func TestCallbackHandler_ErrorStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "denied", err: constants.ErrAccessDenied, wantStatus: http.StatusUnauthorized},
		{name: "conflict", err: constants.ErrAccountLinkedToAnotherUser, wantStatus: http.StatusConflict},
		{name: "validation", err: constants.ErrEmailRequired, wantStatus: http.StatusUnprocessableEntity},
		{name: "provider failure", err: errors.New("internal provider detail"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &callbackUseCaseStub{err: tt.err}
			handler := &handlers.CallbackHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}
			req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/callback/google?code=1&state=1")
			req.SetPathValue("provider", "google")
			req.AddCookie(&http.Cookie{Name: constants.CookieState, Value: "signed-state"})

			handler.Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.False(t, strings.Contains(string(reqCtx.ResponseBody), "internal provider detail"))
		})
	}
}
