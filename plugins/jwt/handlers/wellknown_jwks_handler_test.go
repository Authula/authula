package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/jwt/usecases"
)

type mockJWKSUseCase struct {
	mock.Mock
}

func (m *mockJWKSUseCase) GetJWKS(ctx context.Context) (*usecases.JWKSResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecases.JWKSResult), args.Error(1)
}

func TestWellKnownJWKSHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		prepare        func(*mockJWKSUseCase)
		expectedStatus int
	}{
		{
			name: "use_case_error",
			prepare: func(m *mockJWKSUseCase) {
				m.On("GetJWKS", mock.Anything).Return((*usecases.JWKSResult)(nil), errors.New("no keys")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "success",
			prepare: func(m *mockJWKSUseCase) {
				m.On("GetJWKS", mock.Anything).Return(&usecases.JWKSResult{KeySet: jwk.NewSet()}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUC := &mockJWKSUseCase{}
			if tt.prepare != nil {
				tt.prepare(mockUC)
			}

			handler := &WellKnownJWKSHandler{
				Logger:      &internaltests.MockLogger{},
				JWKSUseCase: mockUC,
			}

			req, w, reqCtx := internaltests.NewHandlerRequestWithActor(t, http.MethodGet, "/.well-known/jwks.json", nil, nil)
			handler.Handle().ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, reqCtx.ResponseStatus)
			mockUC.AssertExpectations(t)
		})
	}
}
