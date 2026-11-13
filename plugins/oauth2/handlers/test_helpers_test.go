package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Authula/authula/models"
)

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
