package constants

import "errors"

// Event type constants
const (
	EventTokenReuseRecovered = "jwt.token.reuse.recovered"
	EventTokenReuseThrottled = "jwt.token.reuse.throttled"
	EventTokenReuseMalicious = "jwt.token.reuse.malicious"
)

// Error definitions
var (
	ErrInvalidToken          = errors.New("provided token is invalid or malformed")
	ErrTokenExpired          = errors.New("token has expired")
	ErrTokenRevoked          = errors.New("token has been revoked")
	ErrRefreshTokenNotFound  = errors.New("refresh token not found")
	ErrJWKSNotFound          = errors.New("JWKS (JSON Web Key Set) not available")
	ErrTokenGenerationFailed = errors.New("failed to generate token")
	ErrTokenValidationFailed = errors.New("failed to validate token")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionExpired        = errors.New("session expired")
	ErrInvalidSessionState   = errors.New("session is in invalid state")
)
