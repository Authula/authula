package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestJWKSUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*useCaseTestFixture)
		assert func(*testing.T, *JWKSResult, error)
	}{
		{
			name: "success",
			setup: func(f *useCaseTestFixture) {
				set := jwk.NewSet()
				f.cacheSvc.On("GetJWKSWithFallback", mock.Anything).Return(set, nil).Once()
			},
			assert: func(t *testing.T, result *JWKSResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result.KeySet)
			},
		},
		{
			name: "service_error",
			setup: func(f *useCaseTestFixture) {
				f.cacheSvc.On("GetJWKSWithFallback", mock.Anything).Return(jwk.Set(nil), errors.New("cache error")).Once()
			},
			assert: func(t *testing.T, result *JWKSResult, err error) {
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
			result, err := f.newJWKSUseCase().GetJWKS(context.Background())
			tc.assert(t, result, err)
			f.cacheSvc.AssertExpectations(t)
		})
	}
}
