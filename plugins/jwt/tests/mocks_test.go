package tests

import (
	"github.com/Authula/authula/plugins/jwt/repositories"
	jwtservices "github.com/Authula/authula/plugins/jwt/services"
	"github.com/Authula/authula/services"
)

var _ jwtservices.TokenService = (*MockTokenService)(nil)
var _ jwtservices.SigningKeyService = (*MockSigningKeyService)(nil)
var _ jwtservices.SigningKeyService = (*MockTokenService)(nil)
var _ jwtservices.KeyService = (*MockKeyService)(nil)
var _ jwtservices.CacheService = (*MockCacheService)(nil)
var _ jwtservices.BlacklistService = (*MockBlacklistService)(nil)
var _ jwtservices.RefreshTokenRepository = (*MockRefreshTokenRepository)(nil)
var _ jwtservices.RefreshTokenService = (*MockRefreshTokenService)(nil)
var _ services.JWTService = (*MockJWTService)(nil)
var _ services.JWTService = (*MockTokenService)(nil)
var _ services.SessionService = (*MockSessionService)(nil)
var _ services.TokenService = (*MockTokenServiceCore)(nil)
var _ repositories.SigningKeyRepository = (*MockSigningKeyRepository)(nil)
var _ interface{ Encrypt(string) (string, error) } = (*MockEncryptor)(nil)
