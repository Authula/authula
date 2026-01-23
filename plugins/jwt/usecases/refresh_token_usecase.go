package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/services"
)

type RefreshTokenUseCaseImpl struct {
	Logger  models.Logger
	Service services.RefreshTokenService
}

func NewRefreshTokenUseCase(
	logger models.Logger,
	service services.RefreshTokenService,
) RefreshTokenUseCase {
	return &RefreshTokenUseCaseImpl{
		Logger:  logger,
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
	}, nil
}
