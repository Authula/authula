package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/jwt/services"
)

type RefreshTokenUseCaseImpl struct {
	Service services.RefreshTokenService
}

func NewRefreshTokenUseCase(
	service services.RefreshTokenService,
) RefreshTokenUseCase {
	return &RefreshTokenUseCaseImpl{
		Service: service,
	}
}

func (uc *RefreshTokenUseCaseImpl) RefreshTokens(ctx context.Context, refreshToken string) (*RefreshTokenResult, error) {
	result, err := uc.Service.RefreshTokens(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResult{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
	}, nil
}
