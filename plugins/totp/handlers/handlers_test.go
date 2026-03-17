package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

func newReqCtx(t *testing.T, method, path string, body []byte, userID *string) (*http.Request, *models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Values:         make(map[string]any),
		UserID:         userID,
	}
	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)
	reqCtx.Request = req
	return req, reqCtx, w
}

func TestEnableHandler_Unauthenticated(t *testing.T) {
	h := &EnableHandler{}
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/enable", []byte(`{"password":"x"}`), nil)
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
}

func TestEnableHandler_InvalidBody(t *testing.T) {
	uid := "u1"
	h := &EnableHandler{}
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/enable", []byte("not-json"), &uid)
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, reqCtx.ResponseStatus)
}

func TestDisableHandler_Unauthenticated(t *testing.T) {
	h := &DisableHandler{}
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/disable", []byte(`{"password":"x"}`), nil)
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
}

func TestGetTOTPURIHandler_Unauthenticated(t *testing.T) {
	h := &GetTOTPURIHandler{}
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/get-uri", []byte(`{"password":"x"}`), nil)
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
}

func TestVerifyTOTPHandler_MissingPendingCookie(t *testing.T) {
	h := &VerifyTOTPHandler{PluginConfig: &types.TOTPPluginConfig{}}
	uid := "u1"
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/verify", []byte(`{"code":"123456"}`), &uid)
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
}

func TestVerifyTOTPHandler_InvalidBody(t *testing.T) {
	h := &VerifyTOTPHandler{PluginConfig: &types.TOTPPluginConfig{}}
	uid := "u1"
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/verify", []byte("bad"), &uid)
	req.AddCookie(&http.Cookie{Name: "totp_pending", Value: "token"})
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, reqCtx.ResponseStatus)
}

func TestVerifyBackupCodeHandler_MissingPendingCookie(t *testing.T) {
	h := &VerifyBackupCodeHandler{PluginConfig: &types.TOTPPluginConfig{}}
	uid := "u1"
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/verify-backup-code", []byte(`{"code":"abc"}`), &uid)
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, reqCtx.ResponseStatus)
}

func TestVerifyBackupCodeHandler_InvalidBody(t *testing.T) {
	h := &VerifyBackupCodeHandler{PluginConfig: &types.TOTPPluginConfig{}}
	uid := "u1"
	req, reqCtx, w := newReqCtx(t, http.MethodPost, "/totp/verify-backup-code", []byte("bad"), &uid)
	req.AddCookie(&http.Cookie{Name: "totp_pending", Value: "token"})
	h.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, reqCtx.ResponseStatus)
}
