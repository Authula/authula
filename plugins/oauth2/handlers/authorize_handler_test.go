package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	called bool
}

func (s *authorizeUseCaseStub) Authorize(_ context.Context, req *types.AuthorizeRequest) (*usecases.AuthorizeResult, error) {
	s.called = true
	s.req = req
	return s.result, s.err
}

func TestAuthorizeHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		result           *usecases.AuthorizeResult
		err              error
		actor            *models.Actor
		provider         string
		wantStatus       int
		wantActorID      string
		wantRedirectTo   string
		wantBodyContains string
		wantCalled       bool
		checkCookies     func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success sets the binding cookie",
			result: &usecases.AuthorizeResult{
				AuthorizationURL: "https://provider.example/authorize?state=1",
				Binding:          "binding-1",
			},
			actor:          &models.Actor{ID: "user-1", Type: models.ActorUser},
			provider:       "google",
			wantStatus:     http.StatusOK,
			wantActorID:    "user-1",
			wantRedirectTo: "/welcome",
			wantCalled:     true,
			checkCookies: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				bindingCookie := findCookie(t, recorder, constants.CookieFlowBinding)
				require.NotNil(t, bindingCookie)
				assert.Equal(t, "binding-1", bindingCookie.Value)
				assert.Equal(t, "api.example.com", bindingCookie.Domain)
				assert.False(t, bindingCookie.Secure, "Secure must follow config, not X-Forwarded-Proto")
				assert.True(t, bindingCookie.HttpOnly)
				assert.Equal(t, http.SameSiteLaxMode, bindingCookie.SameSite)
			},
		},
		{
			name:             "error maps without leaking details",
			err:              constants.ErrProviderNotFound,
			provider:         "missing",
			wantStatus:       http.StatusNotFound,
			wantBodyContains: "provider not found",
			wantCalled:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := &authorizeUseCaseStub{result: tt.result, err: tt.err}
			handler := &handlers.AuthorizeHandler{
				UseCase: stub,
				Cookie: handlers.CookieOptions{
					Secure: false,
					Domain: "api.example.com",
					MaxAge: 5 * time.Minute,
				},
				Logger: &internaltests.MockLogger{},
			}

			target := "/oauth2/authorize/" + tt.provider
			if tt.wantRedirectTo != "" {
				target += "?redirect_to=" + tt.wantRedirectTo
			}
			req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, target)
			req.SetPathValue("provider", tt.provider)
			req.Header.Set("X-Forwarded-Proto", "https")
			reqCtx.Actor = tt.actor

			handler.Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
			assert.Equal(t, tt.wantCalled, stub.called)
			if tt.wantBodyContains != "" {
				assert.Contains(t, string(reqCtx.ResponseBody), tt.wantBodyContains)
			}
			if tt.wantCalled {
				require.NotNil(t, stub.req)
				assert.Equal(t, tt.provider, stub.req.ProviderID)
				assert.Equal(t, tt.wantActorID, stub.req.ActorID)
				assert.Equal(t, tt.wantRedirectTo, stub.req.RedirectTo)
			}
			if tt.checkCookies != nil {
				tt.checkCookies(t, recorder)
			}
		})
	}
}
