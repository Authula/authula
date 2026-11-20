package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/jwt/types"
	"github.com/Authula/authula/plugins/jwt/usecases"
)

type mockRefreshTokenUseCase struct {
	mock.Mock
}

// panickingRefreshTokenUseCase simulates an unexpected failure inside the use case so the
// handler's deferred response handling can be observed while the panic unwinds.
type panickingRefreshTokenUseCase struct {
	message string
}

func (u *panickingRefreshTokenUseCase) RefreshTokens(context.Context, string) (*usecases.RefreshTokenResult, error) {
	panic(u.message)
}

func (m *mockRefreshTokenUseCase) RefreshTokens(ctx context.Context, refreshToken string) (*usecases.RefreshTokenResult, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecases.RefreshTokenResult), args.Error(1)
}

func TestRefreshTokenHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           []byte
		prepare        func(*mockRefreshTokenUseCase)
		expectedStatus int
		wantBody       string
	}{
		{
			name:           "invalid_json",
			body:           []byte("{"),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "missing_refresh_token",
			body:           internaltests.MarshalToJSON(t, types.RefreshTokenRequest{RefreshToken: ""}),
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "use_case_error",
			body: internaltests.MarshalToJSON(t, types.RefreshTokenRequest{RefreshToken: "bad-token"}),
			prepare: func(m *mockRefreshTokenUseCase) {
				m.On("RefreshTokens", mock.Anything, "bad-token").Return((*usecases.RefreshTokenResult)(nil), errors.New("internal detail")).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			wantBody:       `{"message":"invalid refresh token"}`,
		},
		{
			name: "success",
			body: internaltests.MarshalToJSON(t, types.RefreshTokenRequest{RefreshToken: "valid-token"}),
			prepare: func(m *mockRefreshTokenUseCase) {
				m.On("RefreshTokens", mock.Anything, "valid-token").Return(&usecases.RefreshTokenResult{
					AccessToken:  "new-access",
					RefreshToken: "new-refresh",
					TokenType:    "Bearer",
					ExpiresIn:    900,
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			wantBody:       `{"access_token":"new-access","refresh_token":"new-refresh","token_type":"Bearer","expires_in":900}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockRefreshTokenUseCase{}
			if tt.prepare != nil {
				tt.prepare(mockUC)
			}

			handler := &RefreshTokenHandler{
				RefreshTokenUseCase: mockUC,
			}

			req, w, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodPost, "/token/refresh", tt.body, nil)
			handler.Handle().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			require.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
			require.Equal(t, "no-cache", reqCtx.ResponseHeaders.Get("Pragma"))
			require.NotContains(t, string(reqCtx.ResponseBody), "internal detail")
			if tt.wantBody != "" {
				require.JSONEq(t, tt.wantBody, string(reqCtx.ResponseBody))
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestRefreshTokenHandler_RequiresRequestContext(t *testing.T) {
	t.Parallel()

	mockUC := &mockRefreshTokenUseCase{}
	handler := &RefreshTokenHandler{
		RefreshTokenUseCase: mockUC,
	}

	// No RequestContext is attached to the request, so the deferred cleanup must not dereference it.
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", strings.NewReader("{"))
	w := httptest.NewRecorder()

	require.Panics(t, func() {
		handler.Handle().ServeHTTP(w, req)
	})
	mockUC.AssertNotCalled(t, "RefreshTokens", mock.Anything, mock.Anything)
	mockUC.AssertExpectations(t)
}

func TestRefreshTokenHandler_InitializesResponseHeadersOnPanic(t *testing.T) {
	t.Parallel()

	handler := &RefreshTokenHandler{
		RefreshTokenUseCase: &panickingRefreshTokenUseCase{message: "refresh use case exploded"},
	}

	body := internaltests.MarshalToJSON(t, types.RefreshTokenRequest{RefreshToken: "valid-token"})
	req, w, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodPost, "/token/refresh", body, nil)
	require.Nil(t, reqCtx.ResponseHeaders)

	require.PanicsWithValue(t, "refresh use case exploded", func() {
		handler.Handle().ServeHTTP(w, req)
	})

	// The deferred cleanup allocates missing headers so the no-store hints are never lost.
	require.Equal(t, "no-store", reqCtx.ResponseHeaders.Get("Cache-Control"))
	require.Equal(t, "no-cache", reqCtx.ResponseHeaders.Get("Pragma"))
	require.False(t, reqCtx.Handled)
}
