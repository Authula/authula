package usecases

import (
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

type useCaseTestFixture struct {
	refreshTokenSvc *jwttests.MockRefreshTokenService
	cacheSvc        *jwttests.MockCacheService
}

func newUseCaseTestFixture() *useCaseTestFixture {
	return &useCaseTestFixture{
		refreshTokenSvc: &jwttests.MockRefreshTokenService{},
		cacheSvc:        &jwttests.MockCacheService{},
	}
}

func (f *useCaseTestFixture) newRefreshTokenUseCase() RefreshTokenUseCase {
	return NewRefreshTokenUseCase(f.refreshTokenSvc)
}

func (f *useCaseTestFixture) newJWKSUseCase() JWKSUseCase {
	return NewJWKSUseCase(f.cacheSvc)
}
