package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
)

type mockVerifyUseCase struct {
	mock.Mock
}

func (m *mockVerifyUseCase) Verify(ctx context.Context, token string, ipAddress *string, userAgent *string) (string, error) {
	args := m.Called(ctx, token, ipAddress, userAgent)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func TestVerifyHandler_RedirectsWithCode(t *testing.T) {
	useCase := &mockVerifyUseCase{}
	useCase.On("Verify", mock.Anything, "token-123", mock.Anything, mock.Anything).Return("token-123", nil).Once()

	handler := &VerifyHandler{
		UseCase:        useCase,
		TrustedOrigins: []string{"https://app.example.com"},
	}

	params := url.Values{}
	params.Set("token", "token-123")
	params.Set("callback_url", "https://app.example.com/welcome")
	req, reqCtx, w := newCallbackRequest(t, "/magic-link/verify?"+params.Encode())

	handler.Handler()(w, req)

	if reqCtx.RedirectURL != "https://app.example.com/welcome?token=token-123" {
		t.Fatalf("expected redirect with token, got %q", reqCtx.RedirectURL)
	}
	if reqCtx.ResponseStatus != http.StatusFound {
		t.Fatalf("expected 302 status, got %d", reqCtx.ResponseStatus)
	}
	if !reqCtx.Handled {
		t.Fatal("expected request to be handled after redirect")
	}
	useCase.AssertExpectations(t)
}

func TestVerifyHandler_ReturnsJSONCode(t *testing.T) {
	useCase := &mockVerifyUseCase{}
	useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("token-123", nil).Once()

	handler := &VerifyHandler{UseCase: useCase}
	req, reqCtx, w := newCallbackRequest(t, "/magic-link/verify?token=abc")

	handler.Handler()(w, req)

	if reqCtx.ResponseStatus != http.StatusOK {
		t.Fatalf("expected status OK, got %d", reqCtx.ResponseStatus)
	}

	var resp types.VerifyResponse
	if err := json.Unmarshal(reqCtx.ResponseBody, &resp); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	if resp.Token != "token-123" {
		t.Fatalf("expected token 'token-123', got %q", resp.Token)
	}
	useCase.AssertExpectations(t)
}

func TestVerifyHandler_MissingToken(t *testing.T) {
	handler := &VerifyHandler{UseCase: &mockVerifyUseCase{}}
	req, reqCtx, w := newCallbackRequest(t, "/magic-link/verify")

	handler.Handler()(w, req)

	assertErrorResponse(t, reqCtx, http.StatusBadRequest, "token is required")
}

func TestVerifyHandler_InvalidCallbackURL(t *testing.T) {
	useCase := &mockVerifyUseCase{}
	useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("token-123", nil).Once()
	handler := &VerifyHandler{UseCase: useCase}
	params := url.Values{}
	params.Set("token", "abc")
	params.Set("callback_url", "ht!tp://bad url")
	req, reqCtx, w := newCallbackRequest(t, "/magic-link/verify?"+params.Encode())

	handler.Handler()(w, req)

	assertErrorResponse(t, reqCtx, http.StatusBadRequest, "invalid callback_url")
	useCase.AssertExpectations(t)
}

func TestVerifyHandler_UntrustedCallbackURL(t *testing.T) {
	useCase := &mockVerifyUseCase{}
	useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("token-123", nil).Once()
	handler := &VerifyHandler{
		UseCase:        useCase,
		TrustedOrigins: []string{"https://trusted.com"},
	}
	req, reqCtx, w := newCallbackRequest(t, "/magic-link/verify?token=abc&callback_url=https://evil.com")

	handler.Handler()(w, req)

	assertErrorResponse(t, reqCtx, http.StatusBadRequest, "callback_url is not a trusted origin")
	useCase.AssertExpectations(t)
}

func TestVerifyHandler_UseCaseError(t *testing.T) {
	useCase := &mockVerifyUseCase{}
	useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("", errors.New("boom")).Once()

	handler := &VerifyHandler{UseCase: useCase}
	req, reqCtx, w := newCallbackRequest(t, "/magic-link/verify?token=abc")

	handler.Handler()(w, req)

	assertErrorResponse(t, reqCtx, http.StatusBadRequest, "boom")
	useCase.AssertExpectations(t)
}

func newCallbackRequest(t *testing.T, target string) (*http.Request, *models.RequestContext, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
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

func assertErrorResponse(t *testing.T, reqCtx *models.RequestContext, status int, message string) {
	t.Helper()
	if !reqCtx.Handled {
		t.Fatal("expected request to be handled")
	}
	if reqCtx.ResponseStatus != status {
		t.Fatalf("expected status %d, got %d", status, reqCtx.ResponseStatus)
	}
	var body map[string]any
	if err := json.Unmarshal(reqCtx.ResponseBody, &body); err != nil {
		t.Fatalf("expected JSON body, got error: %v", err)
	}
	if body["message"] != message {
		t.Fatalf("expected message %q, got %v", message, body["message"])
	}
}
