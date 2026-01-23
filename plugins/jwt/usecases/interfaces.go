package usecases

import (
	"context"
)

type RefreshTokenUseCase interface {
	RefreshTokens(ctx context.Context, refreshToken string) (*RefreshTokenResult, error)
}

type JWKSUseCase interface {
	GetJWKS(ctx context.Context) (*JWKSResult, error)
}
