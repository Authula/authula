package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
)

func TestExchangeHandler_SuccessSetsSessionContext(t *testing.T) {
	user := &models.User{ID: "user-123", Email: "user@example.com"}
	session := &models.Session{ID: "sess-456", UserID: "user-123"}
	useCase := &stubExchangeUseCase{
		result: &types.ExchangeResult{
			User:         user,
			Session:      session,
			SessionToken: "session-token",
		},
	}

	handler := &ExchangeHandler{UseCase: useCase}
	body := bytes.NewBufferString(`{"token":"token-123"}`)
	req, reqCtx, w := newExchangeRequest(t, body)

	handler.Handler()(w, req)

	if reqCtx.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status OK, got %d", reqCtx.ResponseStatus)
	}
	if reqCtx.UserID == nil || *reqCtx.UserID != "user-123" {
		t.Fatalf("expected user id to be set, got %v", reqCtx.UserID)
	}
	if reqCtx.Values[models.ContextSessionID.String()] != "sess-456" {
		t.Fatalf("expected session id in context, got %v", reqCtx.Values[models.ContextSessionID.String()])
	}
	if reqCtx.Values[models.ContextSessionToken.String()] != "session-token" {
		t.Fatalf("expected session token value, got %v", reqCtx.Values[models.ContextSessionToken.String()])
	}
	if reqCtx.Values[models.ContextAuthSuccess.String()] != true {
		t.Fatal("expected auth success flag")
	}

	var resp types.ExchangeResponse
	if err := json.Unmarshal(reqCtx.ResponseBody, &resp); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	if resp.User == nil || resp.User.ID != user.ID {
		t.Fatalf("expected user in response, got %v", resp.User)
	}
	if resp.Session == nil || resp.Session.ID != session.ID {
		t.Fatalf("expected session in response, got %v", resp.Session)
	}
	if useCase.lastToken != "token-123" {
		t.Fatalf("expected token to be passed to use case, got %s", useCase.lastToken)
	}
}

func TestExchangeHandler_MissingToken(t *testing.T) {
	handler := &ExchangeHandler{UseCase: &stubExchangeUseCase{}}
	req, reqCtx, w := newExchangeRequest(t, bytes.NewBufferString("{}"))

	handler.Handler()(w, req)

	assertErrorResponse(t, reqCtx, http.StatusBadRequest, "token is required")
}

func TestExchangeHandler_InvalidJSON(t *testing.T) {
	handler := &ExchangeHandler{UseCase: &stubExchangeUseCase{}}
	req, reqCtx, w := newExchangeRequest(t, bytes.NewBufferString("{invalid"))

	handler.Handler()(w, req)

	assertErrorResponse(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
}

func TestExchangeHandler_UseCaseError(t *testing.T) {
	handler := &ExchangeHandler{UseCase: &stubExchangeUseCase{err: errors.New("exchange failed")}}
	req, reqCtx, w := newExchangeRequest(t, bytes.NewBufferString(`{"token":"token-123"}`))

	handler.Handler()(w, req)

	assertErrorResponse(t, reqCtx, http.StatusBadRequest, "exchange failed")
}

func TestExchangeHandler_PassesRequestMetadataToUseCase(t *testing.T) {
	user := &models.User{ID: "user-123", Email: "user@example.com"}
	session := &models.Session{ID: "sess-456", UserID: "user-123"}
	useCase := &stubExchangeUseCase{
		result: &types.ExchangeResult{User: user, Session: session, SessionToken: "session-token"},
	}

	handler := &ExchangeHandler{UseCase: useCase}
	body := bytes.NewBufferString(`{"token":"token-123"}`)
	req, reqCtx, w := newExchangeRequest(t, body)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	reqCtx.ClientIP = "127.0.0.1"

	handler.Handler()(w, req)

	if useCase.lastIP == nil || *useCase.lastIP != "127.0.0.1" {
		t.Fatalf("expected IP metadata to be forwarded, got %v", useCase.lastIP)
	}
	if useCase.lastUserAgent == nil || *useCase.lastUserAgent != "TestAgent/1.0" {
		t.Fatalf("expected user agent metadata, got %v", useCase.lastUserAgent)
	}
}

func newExchangeRequest(t *testing.T, body *bytes.Buffer) (*http.Request, *models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/magic-link/exchange", body)
	w := httptest.NewRecorder()
	reqCtx := &models.RequestContext{
		Request:        req,
		ResponseWriter: w,
		Path:           req.URL.Path,
		Method:         req.Method,
		Headers:        req.Header,
		ClientIP:       "127.0.0.1",
		Values:         make(map[string]any),
	}
	ctx := models.SetRequestContext(context.Background(), reqCtx)
	req = req.WithContext(ctx)
	reqCtx.Request = req
	return req, reqCtx, w
}

type stubExchangeUseCase struct {
	result        *types.ExchangeResult
	err           error
	lastToken     string
	lastIP        *string
	lastUserAgent *string
}

func (s *stubExchangeUseCase) Exchange(ctx context.Context, token string, ipAddress *string, userAgent *string) (*types.ExchangeResult, error) {
	s.lastToken = token
	s.lastIP = ipAddress
	s.lastUserAgent = userAgent
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return nil, nil
	}
	return s.result, nil
}
