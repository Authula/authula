package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	plugintests "github.com/Authula/authula/plugins/magic-link/tests"
	"github.com/Authula/authula/plugins/magic-link/types"
)

func TestVerifyHandler_Handler(t *testing.T) {
	testCases := []struct {
		name         string
		target       string
		trustedHosts []string
		setup        func(t *testing.T, useCase *plugintests.MockVerifyUseCase)
		assertResult func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name:         "redirects with code",
			target:       "/magic-link/verify?token=token-123&callback_url=https://app.example.com/welcome",
			trustedHosts: []string{"https://app.example.com"},
			setup: func(t *testing.T, useCase *plugintests.MockVerifyUseCase) {
				useCase.On("Verify", mock.Anything, "token-123", mock.Anything, mock.Anything).Return("token-123", nil).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				if reqCtx.RedirectURL != "https://app.example.com/welcome?token=token-123" {
					t.Fatalf("expected redirect with token, got %q", reqCtx.RedirectURL)
				}
				if reqCtx.ResponseStatus != http.StatusFound {
					t.Fatalf("expected 302 status, got %d", reqCtx.ResponseStatus)
				}
				if !reqCtx.Handled {
					t.Fatal("expected request to be handled after redirect")
				}
			},
		},
		{
			name:   "returns JSON code",
			target: "/magic-link/verify?token=abc",
			setup: func(t *testing.T, useCase *plugintests.MockVerifyUseCase) {
				useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("token-123", nil).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
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
			},
		},
		{
			name:   "missing token",
			target: "/magic-link/verify",
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "token is required")
			},
		},
		{
			name:   "invalid callback url",
			target: "/magic-link/verify?token=abc&callback_url=ht!tp://bad+url",
			setup: func(t *testing.T, useCase *plugintests.MockVerifyUseCase) {
				useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("token-123", nil).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "invalid callback_url")
			},
		},
		{
			name:         "untrusted callback url",
			target:       "/magic-link/verify?token=abc&callback_url=https://evil.com",
			trustedHosts: []string{"https://trusted.com"},
			setup: func(t *testing.T, useCase *plugintests.MockVerifyUseCase) {
				useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("token-123", nil).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "callback_url is not a trusted origin")
			},
		},
		{
			name:   "use case error",
			target: "/magic-link/verify?token=abc",
			setup: func(t *testing.T, useCase *plugintests.MockVerifyUseCase) {
				useCase.On("Verify", mock.Anything, "abc", mock.Anything, mock.Anything).Return("", errors.New("some error")).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "some error")
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &plugintests.MockVerifyUseCase{}
			if tt.setup != nil {
				tt.setup(t, useCase)
			}

			handler := &VerifyHandler{
				UseCase:        useCase,
				TrustedOrigins: tt.trustedHosts,
			}

			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodGet, tt.target, nil, nil)
			handler.Handler()(w, req)

			tt.assertResult(t, reqCtx)
			useCase.AssertExpectations(t)
		})
	}
}
