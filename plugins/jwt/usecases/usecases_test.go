package usecases

import (
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

type useCaseTestFixture struct {
	logger          *mockLogger
	refreshTokenSvc *jwttests.MockRefreshTokenService
	cacheSvc        *jwttests.MockCacheService
}

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any)  {}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}

func newUseCaseTestFixture() *useCaseTestFixture {
	return &useCaseTestFixture{
		logger:          &mockLogger{},
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
