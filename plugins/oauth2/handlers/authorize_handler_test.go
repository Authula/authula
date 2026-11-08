package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		allowed          bool
		count            int
		limiterErr       error
		wantStatus       int
		wantRetryAfter   bool
		wantRemaining    string
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
			allowed:        true,
			count:          1,
			actor:          &models.Actor{ID: "user-1", Type: models.ActorUser},
			provider:       "google",
			wantStatus:     http.StatusOK,
			wantRemaining:  "9",
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
			allowed:          true,
			count:            1,
			provider:         "missing",
			wantStatus:       http.StatusNotFound,
			wantRemaining:    "9",
			wantBodyContains: "provider not found",
			wantCalled:       true,
		},
		{
			name:           "rate limit rejection returns 429",
			allowed:        false,
			count:          11,
			provider:       "google",
			wantStatus:     http.StatusTooManyRequests,
			wantRetryAfter: true,
			wantRemaining:  "0",
			wantCalled:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rateLimiter := &internaltests.MockRateLimitService{}
			resetAt := time.Now().Add(30 * time.Second)
			rateLimiter.On(
				"CheckAndIncrement",
				mock.Anything,
				"oauth2:authorize:127.0.0.1",
				time.Minute,
				10,
			).Return(tt.allowed, tt.count, resetAt, tt.limiterErr).Once()

			stub := &authorizeUseCaseStub{result: tt.result, err: tt.err}
			handler := &handlers.AuthorizeHandler{
				UseCase:         stub,
				RateLimiter:     rateLimiter,
				RateLimitWindow: time.Minute,
				RateLimitMax:    10,
				Cookie: handlers.CookieOptions{
					Secure: false,
					Domain: "api.example.com",
					MaxAge: 5 * time.Minute,
				},
				Logger: &internaltests.MockLogger{},
			}

			target := "/oauth2/authorize/" + tt.provider
			if tt.wantRedirectTo != "" {
				target += "?redirect_to=/welcome"
			}
			req, reqCtx, recorder := newHandlerRequest(t, http.MethodGet, target)
			req.SetPathValue("provider", tt.provider)
			req.Header.Set("X-Forwarded-Proto", "https")
			reqCtx.Actor = tt.actor
			reqCtx.ClientIP = "127.0.0.1"

			handler.Handle()(recorder, req)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
			assert.Equal(t, tt.wantCalled, stub.called)
			if tt.limiterErr == nil {
				assert.Equal(t, "10", recorder.Header().Get("X-RateLimit-Limit"))
				assert.Equal(t, tt.wantRemaining, recorder.Header().Get("X-RateLimit-Remaining"))
				assert.Equal(t, strconv.FormatInt(resetAt.Unix(), 10), recorder.Header().Get("X-RateLimit-Reset"))
			}
			if tt.wantRetryAfter {
				assert.NotEmpty(t, recorder.Header().Get("X-Retry-After"))

				var body map[string]any
				require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &body))
				assert.Equal(t, float64(10), body["limit"])
				assert.Equal(t, float64(0), body["remaining"])
				assert.NotZero(t, body["retry_after"])
			}
			if tt.wantBodyContains != "" {
				assert.Contains(t, string(reqCtx.ResponseBody), tt.wantBodyContains)
			}
			if tt.wantCalled && tt.actor != nil {
				require.NotNil(t, stub.req)
				assert.Equal(t, tt.wantActorID, stub.req.ActorID)
				assert.Equal(t, tt.wantRedirectTo, stub.req.RedirectTo)
			}
			if tt.checkCookies != nil {
				tt.checkCookies(t, recorder)
			}
			rateLimiter.AssertExpectations(t)
		})
	}
}
