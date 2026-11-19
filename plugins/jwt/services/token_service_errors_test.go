package services

import (
	"context"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

type tokenContextKey struct{}

func TestTokenService_GenerateTokensPropagateCallerContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(*tokenService, context.Context) error
	}{
		{
			name: "GenerateUserToken",
			call: func(s *tokenService, ctx context.Context) error {
				_, err := s.GenerateUserToken(ctx, "user-1", "session-1", nil)
				return err
			},
		},
		{
			name: "GenerateMachineToken",
			call: func(s *tokenService, ctx context.Context) error {
				_, err := s.GenerateMachineToken(ctx, "client-1", "", nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			ctx := context.WithValue(context.Background(), tokenContextKey{}, "caller-value")

			f.keySvc.On("GetActiveKey", mock.MatchedBy(func(got context.Context) bool {
				return got.Value(tokenContextKey{}) == "caller-value"
			})).Return(f.activeKey, nil).Once()
			f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil).Once()

			require.NoError(t, tt.call(svc, ctx))
			f.keySvc.AssertExpectations(t)
			f.coreTokenSvc.AssertExpectations(t)
		})
	}
}

func TestTokenService_UnsupportedSigningAlgorithm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(*tokenService, context.Context) error
	}{
		{
			name: "GenerateUserToken",
			call: func(s *tokenService, ctx context.Context) error {
				_, err := s.GenerateUserToken(ctx, "user-1", "session-1", nil)
				return err
			},
		},
		{
			name: "GenerateMachineToken",
			call: func(s *tokenService, ctx context.Context) error {
				_, err := s.GenerateMachineToken(ctx, "client-1", "", nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			unsupported := *f.activeKey
			unsupported.Algorithm = "not-a-real-algorithm"

			f.keySvc.On("GetActiveKey", mock.Anything).Return(&unsupported, nil).Once()
			f.coreTokenSvc.On("Decrypt", unsupported.PrivateKey).Return(unsupported.PrivateKey, nil).Once()

			err := tt.call(svc, context.Background())

			require.ErrorIs(t, err, models.ErrUnsupportedSigningAlgorithm)
			f.keySvc.AssertExpectations(t)
			f.coreTokenSvc.AssertExpectations(t)
		})
	}
}

// setErrorToken embeds a jwt.Token and records Set calls so tests can drive
// branches that depend on the token implementation rejecting a value.
type setErrorToken struct {
	jwt.Token
	err   error
	calls []string
}

func (t *setErrorToken) Set(name string, value any) error {
	t.calls = append(t.calls, name)
	if t.err != nil {
		return t.err
	}
	return t.Token.Set(name, value)
}

func TestTokenService_ApplyExtraClaims(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		claims      map[string]any
		wantErr     error
		wantErrMsg  string
		wantNoCalls bool
	}{
		{
			name:        "reserved claims are ignored",
			claims:      map[string]any{jwt.IssuerKey: "https://evil.example.com", jwt.SubjectKey: "attacker"},
			wantNoCalls: true,
		},
		{
			name:       "non-reserved claims are applied",
			claims:     map[string]any{"custom": "value"},
			wantErr:    assert.AnError,
			wantErrMsg: "failed to set extra claim custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &tokenService{logger: &internaltests.MockLogger{}}
			token := &setErrorToken{Token: jwt.New(), err: assert.AnError}

			err := svc.applyExtraClaims(token, tt.claims)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				require.NoError(t, err)
			}
			if tt.wantNoCalls {
				assert.Empty(t, token.calls)
			}
		})
	}
}
