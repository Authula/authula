package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
)

type SignInUseCase interface {
	SignIn(ctx context.Context, name *string, email string, callbackURL *string) (*types.SignInResult, error)
}

type VerifyUseCase interface {
	Verify(ctx context.Context, token string, ipAddress *string, userAgent *string) (string, error)
}

type ExchangeUseCase interface {
	Exchange(ctx context.Context, token string, ipAddress *string, userAgent *string) (*types.ExchangeResult, error)
}

type UseCases struct {
	SignInUseCase   SignInUseCase
	VerifyUseCase   VerifyUseCase
	ExchangeUseCase ExchangeUseCase
}

func NewUseCases(
	signInUseCase SignInUseCase,
	verifyUseCase VerifyUseCase,
	exchangeUseCase ExchangeUseCase,
) *UseCases {
	return &UseCases{
		SignInUseCase:   signInUseCase,
		VerifyUseCase:   verifyUseCase,
		ExchangeUseCase: exchangeUseCase,
	}
}
