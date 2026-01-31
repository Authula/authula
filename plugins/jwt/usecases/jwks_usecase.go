package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/services"
)

type JWKSUseCaseImpl struct {
	Logger  models.Logger
	Service services.CacheService
}

func NewJWKSUseCase(
	logger models.Logger,
	service services.CacheService,
) JWKSUseCase {
	return &JWKSUseCaseImpl{
		Logger:  logger,
		Service: service,
	}
}

func (uc *JWKSUseCaseImpl) GetJWKS(ctx context.Context) (*JWKSResult, error) {
	keySet, err := uc.Service.GetJWKSWithFallback(ctx)
	if err != nil {
		return nil, err
	}

	return &JWKSResult{
		KeySet: keySet,
	}, nil
}
