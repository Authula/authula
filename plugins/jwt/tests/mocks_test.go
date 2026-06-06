package tests

import (
	jwtservices "github.com/Authula/authula/plugins/jwt/services"
	"github.com/Authula/authula/services"
)

var _ jwtservices.TokenService = (*MockTokenService)(nil)
var _ jwtservices.KeyService = (*MockKeyService)(nil)
var _ jwtservices.CacheService = (*MockCacheService)(nil)
var _ jwtservices.BlacklistService = (*MockBlacklistService)(nil)
var _ jwtservices.RefreshTokenRepository = (*MockRefreshTokenRepository)(nil)
var _ jwtservices.RefreshTokenService = (*MockRefreshTokenService)(nil)
var _ services.JWTService = (*MockJWTService)(nil)
var _ services.JWTService = (*MockTokenService)(nil)
var _ services.SessionService = (*MockSessionService)(nil)
var _ services.TokenService = (*MockTokenServiceCore)(nil)
