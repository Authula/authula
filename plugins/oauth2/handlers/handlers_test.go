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

func TestAuthorizeHandler_SuccessSetsCookiesFromConfig(t *testing.T) {
	t.Parallel()

	stub := &authorizeUseCaseStub{result: &usecases.AuthorizeResult{
		AuthorizationURL: "https://provider.example/authorize?state=1",
		StateCookie:      "signed-state",
		RedirectCookie:   "signed-redirect",
		VerifierCookie:   new("signed-verifier"),
	}}
	handler := &handlers.AuthorizeHandler{
		UseCase: stub,
		Cookie: handlers.CookieOptions{
			Secure:   false,
			Domain:   "api.example.com",
			SameSite: http.SameSiteStrictMode,
			MaxAge:   5 * time.Minute,
		},
		Logger: &internaltests.MockLogger{},
	}

	req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/authorize/google?redirect_to=/welcome")
	req.SetPathValue("provider", "google")
	req.Header.Set("X-Forwarded-Proto", "https")
	reqCtx.Actor = &models.Actor{ID: "user-1", Type: models.ActorUser}

	handler.Handle()(recorder, req)

	assert.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
	assert.Equal(t, "user-1", stub.req.ActorID)
	assert.Equal(t, "/welcome", stub.req.RedirectTo)
	assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))

	stateCookie := findCookie(t, recorder, constants.CookieState)
	require.NotNil(t, stateCookie)
	assert.Equal(t, "signed-state", stateCookie.Value)
	assert.Equal(t, "api.example.com", stateCookie.Domain)
	assert.False(t, stateCookie.Secure, "Secure must follow config, not X-Forwarded-Proto")
	assert.Equal(t, http.SameSiteStrictMode, stateCookie.SameSite)

	verifierCookie := findCookie(t, recorder, constants.CookieVerifier)
	require.NotNil(t, verifierCookie)
	assert.Equal(t, "signed-verifier", verifierCookie.Value)
}

func TestAuthorizeHandler_MapsErrorsWithoutLeakingDetails(t *testing.T) {
	t.Parallel()

	stub := &authorizeUseCaseStub{err: constants.ErrProviderNotFound}
	handler := &handlers.AuthorizeHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}

	req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/authorize/missing")
	req.SetPathValue("provider", "missing")

	handler.Handle()(recorder, req)

	assert.Equal(t, http.StatusNotFound, reqCtx.ResponseStatus)
	assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
	assert.Contains(t, string(reqCtx.ResponseBody), "provider not found")
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

func TestCallbackHandler_SuccessReturnsSafeResponseAndSetsSession(t *testing.T) {
	t.Parallel()

	session := &models.Session{
		ID:        "session-1",
		UserID:    "user-1",
		Token:     "hashed-token",
		ExpiresAt: time.Now().Add(time.Hour),
		IPAddress: new("203.0.113.9"),
		UserAgent: new("secret-agent"),
	}
	stub := &callbackUseCaseStub{result: &types.CallbackResult{
		User:         &models.User{ID: "user-1", Email: "user@example.com"},
		Session:      session,
		SessionToken: "plain-token",
	}}
	handler := &handlers.CallbackHandler{
		UseCase: stub,
		Cookie:  handlers.CookieOptions{Secure: true},
		Logger:  &internaltests.MockLogger{},
	}

	req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/callback/google?code=1&state=1")
	req.SetPathValue("provider", "google")
	req.AddCookie(&http.Cookie{Name: constants.CookieState, Value: "signed-state"})

	handler.Handle()(recorder, req)

	assert.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
	assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
	assert.Equal(t, "user-1", reqCtx.Actor.ID)
	assert.Equal(t, "session-1", reqCtx.Values[models.ContextSessionID.String()])
	assert.Equal(t, "plain-token", reqCtx.Values[models.ContextSessionToken.String()])

	var body map[string]any
	require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &body))
	sessionBody := body["session"].(map[string]any)
	assert.NotContains(t, sessionBody, "token")
	assert.NotContains(t, sessionBody, "ip_address")
	assert.NotContains(t, sessionBody, "user_agent")
	assert.NotContains(t, string(reqCtx.ResponseBody), "plain-token")
	assert.NotContains(t, string(reqCtx.ResponseBody), "hashed-token")
}

func TestCallbackHandler_ReusedSessionDoesNotMintANewToken(t *testing.T) {
	t.Parallel()

	stub := &callbackUseCaseStub{result: &types.CallbackResult{
		User:          &models.User{ID: "user-1"},
		Session:       &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)},
		ReusedSession: true,
	}}
	handler := &handlers.CallbackHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}

	req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/callback/google?code=1&state=1")
	req.SetPathValue("provider", "google")
	req.AddCookie(&http.Cookie{Name: constants.CookieState, Value: "signed-state"})

	handler.Handle()(recorder, req)

	assert.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
	assert.Equal(t, "session-1", reqCtx.Values[models.ContextSessionID.String()])
	assert.NotContains(t, reqCtx.Values, models.ContextSessionToken.String())
}

func TestCallbackHandler_RedirectsOnlyAfterUsecaseValidation(t *testing.T) {
	t.Parallel()

	stub := &callbackUseCaseStub{result: &types.CallbackResult{
		User:       &models.User{ID: "user-1"},
		Session:    &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)},
		RedirectTo: "/welcome",
	}}
	handler := &handlers.CallbackHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}

	req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, "/oauth2/callback/google?code=1&state=1")
	req.SetPathValue("provider", "google")
	req.AddCookie(&http.Cookie{Name: constants.CookieState, Value: "signed-state"})

	handler.Handle()(recorder, req)

	assert.Equal(t, "/welcome", reqCtx.RedirectURL)
	assert.Equal(t, http.StatusFound, reqCtx.ResponseStatus)
	assert.Equal(t, "session-1", reqCtx.Values[models.ContextSessionID.String()])
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
