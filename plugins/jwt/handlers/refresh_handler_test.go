package handlers

import (
	"context"
	"errors"
	"net/http"
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
				m.On("RefreshTokens", mock.Anything, "bad-token").Return((*usecases.RefreshTokenResult)(nil), errors.New("invalid")).Once()
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "success",
			body: internaltests.MarshalToJSON(t, types.RefreshTokenRequest{RefreshToken: "valid-token"}),
			prepare: func(m *mockRefreshTokenUseCase) {
				m.On("RefreshTokens", mock.Anything, "valid-token").Return(&usecases.RefreshTokenResult{
					AccessToken:  "new-access",
					RefreshToken: "new-refresh",
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
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
				Logger:              &internaltests.MockLogger{},
				RefreshTokenUseCase: mockUC,
			}

			req, w, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodPost, "/token/refresh", tt.body, nil)
			handler.Handle().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			mockUC.AssertExpectations(t)
		})
	}
}
