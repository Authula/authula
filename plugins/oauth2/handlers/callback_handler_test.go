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

type callbackUseCaseStub struct {
	result *types.CallbackResult
	err    error
	req    *types.CallbackRequest
	called bool
}

func (s *callbackUseCaseStub) Callback(_ context.Context, req *types.CallbackRequest, _ *string, _ *string) (*types.CallbackResult, error) {
	s.called = true
	s.req = req
	return s.result, s.err
}

func newCallbackHandler(stub *callbackUseCaseStub) *handlers.CallbackHandler {
	return &handlers.CallbackHandler{
		UseCase: stub,
		Logger:  &internaltests.MockLogger{},
	}
}

func newCallbackRequest(t *testing.T, target string, withBinding bool) (*http.Request, *models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()

	req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, target)
	req.SetPathValue("provider", "google")
	if withBinding {
		req.AddCookie(&http.Cookie{Name: constants.CookieFlowBinding, Value: "binding-1"})
	}
	return req, reqCtx, recorder
}

func TestCallbackHandler_RequiresStateAndBinding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  string
		binding bool
	}{
		{name: "missing state", target: "/oauth2/callback/google?code=1", binding: true},
		{name: "missing binding", target: "/oauth2/callback/google?code=1&state=state-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &callbackUseCaseStub{}
			req, reqCtx, recorder := newCallbackRequest(t, tt.target, tt.binding)

			newCallbackHandler(stub).Handle()(recorder, req)

			assert.Equal(t, http.StatusUnprocessableEntity, reqCtx.ResponseStatus)
			assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
			assert.False(t, stub.called)
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
			IPAddress: new("127.0.0.1"),
			UserAgent: new("secret-agent"),
		}
	}

	tests := []struct {
		name             string
		result           func() *types.CallbackResult
		wantStatus       int
		wantRedirectURL  string
		wantSessionToken string
		wantSafeBody     bool
		wantNoToken      bool
		wantCacheControl string
		wantBinding      string
	}{
		{
			name: "success returns a safe response and sets the session",
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
			wantBinding:      "binding-1",
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
			wantBinding:      "binding-1",
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
			wantBinding:     "binding-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &callbackUseCaseStub{result: tt.result()}
			req, reqCtx, recorder := newCallbackRequest(t, "/oauth2/callback/google?code=1&state=state-1", true)

			newCallbackHandler(stub).Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.Equal(t, tt.wantCacheControl, reqCtx.ResponseHeaders.Get("Cache-Control"))
			assert.Equal(t, "session-1", reqCtx.Values[models.ContextSessionID.String()])
			assert.Equal(t, tt.wantRedirectURL, reqCtx.RedirectURL)
			require.NotNil(t, stub.req)
			assert.Equal(t, tt.wantBinding, stub.req.Binding)

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

func TestCallbackHandler_ProviderErrorRedirect(t *testing.T) {
	t.Parallel()

	stub := &callbackUseCaseStub{err: &usecases.RedirectError{
		RedirectTo:  "/welcome",
		Code:        "access_denied",
		Description: "user cancelled",
	}}
	req, reqCtx, recorder := newCallbackRequest(t, "/oauth2/callback/google?code=1&state=state-1", true)

	newCallbackHandler(stub).Handle()(recorder, req)

	assert.Equal(t, http.StatusFound, reqCtx.ResponseStatus)
	assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
	assert.Equal(t, "/welcome?error=access_denied&error_description=user+cancelled", reqCtx.RedirectURL)
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
		{name: "provider failure", err: constants.ErrProviderFailure, wantStatus: http.StatusBadGateway},
		{name: "unexpected", err: errors.New("internal provider detail"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &callbackUseCaseStub{err: tt.err}
			req, reqCtx, recorder := newCallbackRequest(t, "/oauth2/callback/google?code=1&state=state-1", true)

			newCallbackHandler(stub).Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.False(t, strings.Contains(string(reqCtx.ResponseBody), "internal provider detail"))
		})
	}
}
