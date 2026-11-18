package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/magic-link/types"
)

var (
	_ SignInUseCase   = (*SignInUseCaseImpl)(nil)
	_ VerifyUseCase   = (*VerifyUseCaseImpl)(nil)
	_ ExchangeUseCase = (*ExchangeUseCaseImpl)(nil)
)

type stubSignInUseCase struct {
	result *types.MagicLinkSignInResult
	err    error
}

func (s *stubSignInUseCase) SignIn(context.Context, *string, string, *string) (*types.MagicLinkSignInResult, error) {
	return s.result, s.err
}

type stubVerifyUseCase struct {
	code string
	err  error
}

func (s *stubVerifyUseCase) Verify(context.Context, string, *string, *string) (string, error) {
	return s.code, s.err
}

type stubExchangeUseCase struct {
	result *types.MagicLinkExchangeResult
	err    error
}

func (s *stubExchangeUseCase) Exchange(context.Context, string, *string, *string) (*types.MagicLinkExchangeResult, error) {
	return s.result, s.err
}

func TestNewUseCases(t *testing.T) {
	signIn := &stubSignInUseCase{result: &types.MagicLinkSignInResult{Token: "sign-in-token"}}
	verify := &stubVerifyUseCase{code: "exchange-code"}
	exchange := &stubExchangeUseCase{
		result: &types.MagicLinkExchangeResult{SessionToken: "session-token"},
	}

	useCases := NewUseCases(signIn, verify, exchange)

	require.NotNil(t, useCases)
	assert.Same(t, signIn, useCases.SignInUseCase)
	assert.Same(t, verify, useCases.VerifyUseCase)
	assert.Same(t, exchange, useCases.ExchangeUseCase)

	ctx := context.Background()
	signInResult, err := useCases.SignInUseCase.SignIn(ctx, nil, "test@example.com", nil)
	require.NoError(t, err)
	assert.Equal(t, "sign-in-token", signInResult.Token)

	code, err := useCases.VerifyUseCase.Verify(ctx, "token", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "exchange-code", code)

	exchangeResult, err := useCases.ExchangeUseCase.Exchange(ctx, "token", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "session-token", exchangeResult.SessionToken)
}

func TestNewUseCasesPropagatesErrors(t *testing.T) {
	useCases := NewUseCases(
		&stubSignInUseCase{err: errors.New("sign in failed")},
		&stubVerifyUseCase{err: errors.New("verify failed")},
		&stubExchangeUseCase{err: errors.New("exchange failed")},
	)

	ctx := context.Background()

	_, err := useCases.SignInUseCase.SignIn(ctx, nil, "test@example.com", nil)
	assert.EqualError(t, err, "sign in failed")

	_, err = useCases.VerifyUseCase.Verify(ctx, "token", nil, nil)
	assert.EqualError(t, err, "verify failed")

	_, err = useCases.ExchangeUseCase.Exchange(ctx, "token", nil, nil)
	assert.EqualError(t, err, "exchange failed")
}
