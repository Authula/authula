package usecases

import (
	internaltests "github.com/Authula/authula/internal/tests"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

type useCaseTestFixture struct {
	logger          *internaltests.MockLogger
	refreshTokenSvc *jwttests.MockRefreshTokenService
	cacheSvc        *jwttests.MockCacheService
}

func newUseCaseTestFixture() *useCaseTestFixture {
	return &useCaseTestFixture{
		logger:          &internaltests.MockLogger{},
		refreshTokenSvc: &jwttests.MockRefreshTokenService{},
		cacheSvc:        &jwttests.MockCacheService{},
	}
}

func (f *useCaseTestFixture) newRefreshTokenUseCase() RefreshTokenUseCase {
	return NewRefreshTokenUseCase(f.logger, f.refreshTokenSvc)
}

func (f *useCaseTestFixture) newJWKSUseCase() JWKSUseCase {
	return NewJWKSUseCase(f.logger, f.cacheSvc)
}
