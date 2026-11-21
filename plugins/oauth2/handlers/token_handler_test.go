package handlers_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"github.com/Authula/authula/plugins/oauth2/types"
)

type fakeTokenUseCase struct {
	result   *types.TokenResult
	err      error
	requests []*types.TokenRequest
}

func (f *fakeTokenUseCase) IssueToken(_ context.Context, req *types.TokenRequest) (*types.TokenResult, error) {
	f.requests = append(f.requests, req)
	return f.result, f.err
}

func newTokenHandlerFixture(t *testing.T, useCase *fakeTokenUseCase) *handlers.TokenHandler {
	t.Helper()

	return &handlers.TokenHandler{
		UseCase: useCase,
		Logger:  &internaltests.MockLogger{},
	}
}

func newTokenRequest(t *testing.T, body string, headers map[string][]string) (*http.Request, *models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/oauth2/token", strings.NewReader(body))
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
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

func basicCredentials(clientID, clientSecret string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(url.QueryEscape(clientID) + ":" + url.QueryEscape(clientSecret)))
	return "Basic " + encoded
}

func performTokenRequest(t *testing.T, handler *handlers.TokenHandler, body string, headers map[string][]string) (*models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()

	req, reqCtx, recorder := newTokenRequest(t, body, headers)
	handler.Handle()(recorder, req)
	return reqCtx, recorder
}

func decodeTokenError(t *testing.T, reqCtx *models.RequestContext) types.TokenErrorResponse {
	t.Helper()

	var payload types.TokenErrorResponse
	require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &payload))
	return payload
}

func TestTokenHandler_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		headers    map[string][]string
		wantScopes []string
		wantScope  string
		checkReq   func(*testing.T, *types.TokenRequest)
	}{
		{
			name:       "client secret basic",
			body:       "grant_type=client_credentials&scope=read+write",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}, "Authorization": {basicCredentials("client-1", "s3cret")}},
			wantScopes: []string{"read", "write"},
			wantScope:  "read write",
			checkReq: func(t *testing.T, req *types.TokenRequest) {
				t.Helper()
				assert.Equal(t, "client-1", req.ClientID)
				assert.Equal(t, "s3cret", req.ClientSecret)
				require.NotNil(t, req.Scope)
				assert.Equal(t, "read write", *req.Scope)
			},
		},
		{
			name:       "client secret post with charset",
			body:       "grant_type=client_credentials&client_id=client-1&client_secret=s3cret",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded; charset=UTF-8"}},
			wantScopes: []string{},
			wantScope:  "",
			checkReq: func(t *testing.T, req *types.TokenRequest) {
				t.Helper()
				assert.Equal(t, "client-1", req.ClientID)
				assert.Equal(t, "s3cret", req.ClientSecret)
				assert.Nil(t, req.Scope)
			},
		},
		{
			name:       "url encoded basic credentials",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}, "Authorization": {basicCredentials("client:1", "s3 cret&")}},
			wantScopes: []string{"orders:read"},
			wantScope:  "orders:read",
			checkReq: func(t *testing.T, req *types.TokenRequest) {
				t.Helper()
				assert.Equal(t, "client:1", req.ClientID)
				assert.Equal(t, "s3 cret&", req.ClientSecret)
			},
		},
		{
			name:       "query credentials ignored",
			body:       "grant_type=client_credentials&client_id=body-client&client_secret=body-secret",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			wantScopes: []string{},
			wantScope:  "",
			checkReq: func(t *testing.T, req *types.TokenRequest) {
				t.Helper()
				assert.Equal(t, "body-client", req.ClientID)
				assert.Equal(t, "body-secret", req.ClientSecret)
				assert.Nil(t, req.Scope)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := &fakeTokenUseCase{result: &types.TokenResult{
				AccessToken: "access-token",
				TokenType:   "Bearer",
				ExpiresIn:   15 * time.Minute,
				Scopes:      tt.wantScopes,
			}}
			handler := newTokenHandlerFixture(t, useCase)

			reqCtx, _ := performTokenRequest(t, handler, tt.body, tt.headers)

			require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
			assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
			assert.Equal(t, "no-cache", reqCtx.ResponseHeaders.Get("Pragma"))
			assert.Empty(t, reqCtx.ResponseHeaders.Get("WWW-Authenticate"))

			var payload types.TokenResponse
			require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &payload))
			assert.Equal(t, "access-token", payload.AccessToken)
			assert.Equal(t, "Bearer", payload.TokenType)
			assert.Equal(t, int64(900), payload.ExpiresIn)
			assert.Equal(t, tt.wantScope, payload.Scope)

			require.Len(t, useCase.requests, 1)
			tt.checkReq(t, useCase.requests[0])
		})
	}
}

func TestTokenHandler_SuccessResponseScopeIsString(t *testing.T) {
	t.Parallel()

	useCase := &fakeTokenUseCase{result: &types.TokenResult{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		ExpiresIn:   time.Minute,
		Scopes:      []string{"read", "write"},
	}}
	handler := newTokenHandlerFixture(t, useCase)

	reqCtx, _ := performTokenRequest(t, handler,
		"grant_type=client_credentials&client_id=client&client_secret=secret",
		map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
	)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &payload))
	assert.Equal(t, "read write", payload["scope"])
}

func TestTokenHandler_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		headers    map[string][]string
		useCaseErr error
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing content type",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "wrong content type",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{"Content-Type": {"application/json"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "multiple content types",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded", "application/json"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "duplicate parameter",
			body:       "grant_type=client_credentials&grant_type=client_credentials&client_id=c&client_secret=s",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "duplicate scope",
			body:       "grant_type=client_credentials&client_id=c&client_secret=s&scope=read&scope=write",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "both auth methods",
			body:       "grant_type=client_credentials&client_id=c&client_secret=s",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}, "Authorization": {basicCredentials("c", "s")}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "missing grant type",
			body:       "client_id=c&client_secret=s",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_request",
		},
		{
			name:       "unsupported grant type",
			body:       "grant_type=authorization_code&client_id=c&client_secret=s",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "unsupported_grant_type",
		},
		{
			name:       "missing post credentials",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_client",
		},
		{
			name:       "missing post secret",
			body:       "grant_type=client_credentials&client_id=c",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_client",
		},
		{
			name:       "malformed basic",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}, "Authorization": {"Basic !!!"}},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_client",
		},
		{
			name:       "non basic authorization",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}, "Authorization": {"Bearer token"}},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_client",
		},
		{
			name:       "multiple authorization headers",
			body:       "grant_type=client_credentials",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}, "Authorization": {basicCredentials("c", "s"), basicCredentials("c2", "s2")}},
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_client",
		},
		{
			name:       "invalid client from use case",
			body:       "grant_type=client_credentials&client_id=c&client_secret=s",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			useCaseErr: constants.ErrInvalidClient,
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_client",
		},
		{
			name:       "grant not allowed",
			body:       "grant_type=client_credentials&client_id=c&client_secret=s",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			useCaseErr: constants.ErrUnauthorizedClient,
			wantStatus: http.StatusBadRequest,
			wantError:  "unauthorized_client",
		},
		{
			name:       "invalid scope",
			body:       "grant_type=client_credentials&client_id=c&client_secret=s&scope=admin",
			headers:    map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
			useCaseErr: constants.ErrInvalidScope,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid_scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := &fakeTokenUseCase{err: tt.useCaseErr}
			handler := newTokenHandlerFixture(t, useCase)

			reqCtx, _ := performTokenRequest(t, handler, tt.body, tt.headers)

			assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
			assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
			assert.Equal(t, "no-cache", reqCtx.ResponseHeaders.Get("Pragma"))

			payload := decodeTokenError(t, reqCtx)
			assert.Equal(t, tt.wantError, payload.Error)

			if tt.wantStatus == http.StatusUnauthorized {
				assert.Equal(t, `Basic realm="oauth2/token", charset="UTF-8"`, reqCtx.ResponseHeaders.Get("WWW-Authenticate"))
			} else {
				assert.Empty(t, reqCtx.ResponseHeaders.Get("WWW-Authenticate"))
			}
		})
	}
}

func TestTokenHandler_InternalErrorUsesProjectShape(t *testing.T) {
	t.Parallel()

	useCase := &fakeTokenUseCase{err: errors.New("signing failed")}
	handler := newTokenHandlerFixture(t, useCase)

	reqCtx, _ := performTokenRequest(t, handler,
		"grant_type=client_credentials&client_id=c&client_secret=s",
		map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
	)

	require.Equal(t, http.StatusInternalServerError, reqCtx.ResponseStatus)
	assert.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
	assert.Equal(t, "no-cache", reqCtx.ResponseHeaders.Get("Pragma"))

	var payload map[string]any
	require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &payload))
	assert.Equal(t, "internal server error", payload["message"])
	assert.NotContains(t, payload, "error")
}

func TestTokenHandler_OversizedBody(t *testing.T) {
	t.Parallel()

	handler := newTokenHandlerFixture(t, &fakeTokenUseCase{})

	reqCtx, _ := performTokenRequest(t, handler,
		strings.Repeat("a", int(constants.TokenRequestMaxBodyBytes)+1),
		map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
	)

	require.Equal(t, http.StatusBadRequest, reqCtx.ResponseStatus)
	assert.Equal(t, "invalid_request", decodeTokenError(t, reqCtx).Error)
}

func TestTokenHandler_PresentButEmptyScopeIsPassedThrough(t *testing.T) {
	t.Parallel()

	useCase := &fakeTokenUseCase{result: &types.TokenResult{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		ExpiresIn:   time.Minute,
		Scopes:      []string{},
	}}
	handler := newTokenHandlerFixture(t, useCase)

	reqCtx, _ := performTokenRequest(t, handler,
		"grant_type=client_credentials&client_id=c&client_secret=s&scope=",
		map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
	)

	require.Equal(t, http.StatusOK, reqCtx.ResponseStatus)
	require.Len(t, useCase.requests, 1)
	require.NotNil(t, useCase.requests[0].Scope)
	assert.Equal(t, "", *useCase.requests[0].Scope)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(reqCtx.ResponseBody, &payload))
	assert.NotContains(t, payload, "scope")
}
