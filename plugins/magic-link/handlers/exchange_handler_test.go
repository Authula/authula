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

func TestExchangeHandler(t *testing.T) {
	testCases := []struct {
		name         string
		body         []byte
		setup        func(t *testing.T, useCase *plugintests.MockExchangeUseCase)
		assertResult func(t *testing.T, reqCtx *models.RequestContext)
	}{
		{
			name: "success sets session context",
			body: internaltests.MarshalToJSON(t, types.ExchangeRequest{Token: "token-123"}),
			setup: func(t *testing.T, useCase *plugintests.MockExchangeUseCase) {
				user := &models.User{ID: "user-123", Email: "user@example.com"}
				session := &models.Session{ID: "sess-456", UserID: "user-123"}
				useCase.On("Exchange", mock.Anything, "token-123", mock.Anything, mock.Anything).
					Return(&types.ExchangeResult{User: user, Session: session, SessionToken: "session-token"}, nil).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
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
				if resp.User == nil || resp.User.ID != "user-123" {
					t.Fatalf("expected user in response, got %v", resp.User)
				}
				if resp.Session == nil || resp.Session.ID != "sess-456" {
					t.Fatalf("expected session in response, got %v", resp.Session)
				}
			},
		},
		{
			name: "missing token",
			body: []byte("{}"),
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "token is required")
			},
		},
		{
			name: "invalid json",
			body: []byte("{invalid"),
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusUnprocessableEntity, "invalid request body")
			},
		},
		{
			name: "use case error",
			body: internaltests.MarshalToJSON(t, types.ExchangeRequest{Token: "token-123"}),
			setup: func(t *testing.T, useCase *plugintests.MockExchangeUseCase) {
				useCase.On("Exchange", mock.Anything, "token-123", mock.Anything, mock.Anything).
					Return(nil, errors.New("exchange failed")).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				internaltests.AssertErrorMessage(t, reqCtx, http.StatusBadRequest, "exchange failed")
			},
		},
		{
			name: "passes request metadata to use case",
			body: internaltests.MarshalToJSON(t, types.ExchangeRequest{Token: "token-123"}),
			setup: func(t *testing.T, useCase *plugintests.MockExchangeUseCase) {
				user := &models.User{ID: "user-123", Email: "user@example.com"}
				session := &models.Session{ID: "sess-456", UserID: "user-123"}
				useCase.On("Exchange", mock.Anything, "token-123", mock.AnythingOfType("*string"), mock.AnythingOfType("*string")).
					Run(func(args mock.Arguments) {
						ip := args.Get(2).(*string)
						userAgent := args.Get(3).(*string)
						if ip == nil || *ip != "127.0.0.1" {
							t.Fatalf("expected IP metadata to be forwarded, got %v", ip)
						}
						if userAgent == nil || *userAgent != "TestAgent/1.0" {
							t.Fatalf("expected user agent metadata, got %v", userAgent)
						}
					}).
					Return(&types.ExchangeResult{User: user, Session: session, SessionToken: "session-token"}, nil).Once()
			},
			assertResult: func(t *testing.T, reqCtx *models.RequestContext) {
				if reqCtx.ResponseStatus != http.StatusOK {
					t.Fatalf("expected status OK, got %d", reqCtx.ResponseStatus)
				}
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			useCase := &plugintests.MockExchangeUseCase{}
			if tt.setup != nil {
				tt.setup(t, useCase)
			}

			handler := &ExchangeHandler{UseCase: useCase}
			req, w, reqCtx := internaltests.NewHandlerRequest(t, http.MethodPost, "/magic-link/exchange", tt.body, nil)

			if tt.name == "passes request metadata to use case" {
				req.Header.Set("User-Agent", "TestAgent/1.0")
				reqCtx.ClientIP = "127.0.0.1"
			}

			handler.Handler()(w, req)

			tt.assertResult(t, reqCtx)
			useCase.AssertExpectations(t)
		})
	}
}
