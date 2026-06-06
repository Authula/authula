package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/jwt/types"
)

func TestRefreshTokenUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		token  string
		setup  func(*useCaseTestFixture)
		assert func(*testing.T, *RefreshTokenResult, error)
	}{
		{
			name:  "success",
			token: "valid-token",
			setup: func(f *useCaseTestFixture) {
				f.refreshTokenSvc.On("RefreshTokens", mock.Anything, "valid-token").Return(&types.RefreshTokenResponse{
					AccessToken:  "new-access",
					RefreshToken: "new-refresh",
				}, nil).Once()
			},
			assert: func(t *testing.T, result *RefreshTokenResult, err error) {
				require.NoError(t, err)
				require.Equal(t, "new-access", result.AccessToken)
				require.Equal(t, "new-refresh", result.RefreshToken)
			},
		},
		{
			name:  "service_error",
			token: "bad-token",
			setup: func(f *useCaseTestFixture) {
				f.refreshTokenSvc.On("RefreshTokens", mock.Anything, "bad-token").Return((*types.RefreshTokenResponse)(nil), errors.New("invalid refresh token")).Once()
			},
			assert: func(t *testing.T, result *RefreshTokenResult, err error) {
				require.Error(t, err)
				require.Nil(t, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newUseCaseTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			result, err := f.newRefreshTokenUseCase().RefreshTokens(context.Background(), tc.token)
			tc.assert(t, result, err)
			f.refreshTokenSvc.AssertExpectations(t)
		})
	}
}
