package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/handlers"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
)

const callbackHandlerTestProvider = "apple"

func newCallbackPostRequest(t *testing.T, target, body string) (*http.Request, *models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func newSignedStateCookie(t *testing.T, state string) *http.Cookie {
	t.Helper()

	signed, err := services.SignCookie(state, []byte("test-hmac-key"))
	require.NoError(t, err)

	return &http.Cookie{Name: constants.CookieState, Value: signed}
}

func successfulCallbackResult() *types.CallbackResult {
	return &types.CallbackResult{
		User:         &models.User{ID: "user-1", Email: "ada@example.com"},
		Session:      &models.Session{ID: "session-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)},
		SessionToken: "plain-token",
	}
}

func TestCallbackHandler_FormPost(t *testing.T) {
	t.Parallel()

	t.Run("form body reaches the usecase with code, state and user payload", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)
		must := require.New(t)

		stub := &callbackUseCaseStub{result: successfulCallbackResult()}
		handler := &handlers.CallbackHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}

		stateCookie := newSignedStateCookie(t, "xyz")
		form := url.Values{
			"code":  {"abc"},
			"state": {"xyz"},
			"user":  {`{"name":{"firstName":"Ada","lastName":"Lovelace"},"email":"attacker@example.com"}`},
		}
		req, reqCtx, recorder := newCallbackPostRequest(
			t,
			"/oauth2/callback/"+callbackHandlerTestProvider,
			form.Encode(),
		)
		req.SetPathValue("provider", callbackHandlerTestProvider)
		req.AddCookie(stateCookie)

		handler.Handle()(recorder, req)

		is.Equal(http.StatusOK, reqCtx.ResponseStatus)
		must.NotNil(stub.req)
		is.Equal(callbackHandlerTestProvider, stub.req.ProviderID)
		is.Equal("abc", stub.req.Code)
		is.Equal("xyz", stub.req.State)
		is.Equal(stateCookie.Value, stub.req.StateCookie, "the signed cookie must reach the usecase verbatim")

		var payload struct {
			Name struct {
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
			} `json:"name"`
			Email string `json:"email"`
		}
		must.NoError(json.Unmarshal([]byte(stub.req.User), &payload))
		is.Equal("Ada", payload.Name.FirstName)
		is.Equal("Lovelace", payload.Name.LastName)
		is.Equal("attacker@example.com", payload.Email, "the handler must forward the payload untouched")

		is.Equal("user-1", reqCtx.Actor.ID)
		is.Equal("session-1", reqCtx.Values[models.ContextSessionID.String()])
		is.Equal("plain-token", reqCtx.Values[models.ContextSessionToken.String()])
		is.Equal("no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
	})

	t.Run("query string is still accepted", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)
		must := require.New(t)

		stub := &callbackUseCaseStub{result: successfulCallbackResult()}
		handler := &handlers.CallbackHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}

		req, reqCtx, recorder := newHandlerRequest(
			t,
			http.MethodGet,
			"/oauth2/callback/"+callbackHandlerTestProvider+"?code=abc&state=xyz",
		)
		req.SetPathValue("provider", callbackHandlerTestProvider)
		req.AddCookie(newSignedStateCookie(t, "xyz"))

		handler.Handle()(recorder, req)

		is.Equal(http.StatusOK, reqCtx.ResponseStatus)
		must.NotNil(stub.req)
		is.Equal("abc", stub.req.Code)
		is.Equal("xyz", stub.req.State)
		is.Empty(stub.req.User)
	})

	t.Run("provider error in the form body reaches the usecase", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)
		must := require.New(t)

		stub := &callbackUseCaseStub{err: constants.ErrAccessDenied}
		handler := &handlers.CallbackHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}

		req, reqCtx, recorder := newCallbackPostRequest(
			t,
			"/oauth2/callback/"+callbackHandlerTestProvider,
			"error=access_denied&state=xyz",
		)
		req.SetPathValue("provider", callbackHandlerTestProvider)
		req.AddCookie(newSignedStateCookie(t, "xyz"))

		handler.Handle()(recorder, req)

		is.Equal(http.StatusUnauthorized, reqCtx.ResponseStatus)
		must.NotNil(stub.req)
		is.Equal("access_denied", stub.req.Error)
		is.Empty(stub.req.Code)
	})

	t.Run("malformed form body maps to ErrInvalidCallback", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		stub := &callbackUseCaseStub{result: successfulCallbackResult()}
		handler := &handlers.CallbackHandler{UseCase: stub, Logger: &internaltests.MockLogger{}}

		req, reqCtx, recorder := newCallbackPostRequest(
			t,
			"/oauth2/callback/"+callbackHandlerTestProvider,
			"code=%zz&state=xyz",
		)
		req.SetPathValue("provider", callbackHandlerTestProvider)
		req.AddCookie(newSignedStateCookie(t, "xyz"))

		handler.Handle()(recorder, req)

		is.Equal(http.StatusUnprocessableEntity, reqCtx.ResponseStatus)
		is.Contains(string(reqCtx.ResponseBody), "invalid request")
		is.True(reqCtx.Handled)
		is.Nil(stub.req, "a malformed body must not reach the usecase")

		for _, name := range []string{constants.CookieState, constants.CookieRedirectTo, constants.CookieVerifier} {
			cookie := findCookie(t, recorder, name)
			require.NotNil(t, cookie, "expected a clearing cookie for %s", name)
			is.Less(cookie.MaxAge, 0)
		}
	})
}
