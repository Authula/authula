package usecases

import (
	"context"

	"github.com/Authula/authula/plugins/jwt/services"
)

type JWKSUseCaseImpl struct {
	Service services.CacheService
}

func NewJWKSUseCase(
	service services.CacheService,
) JWKSUseCase {
	return &JWKSUseCaseImpl{
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
