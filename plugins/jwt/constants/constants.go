package constants

import "errors"

const (
	EventTokenReuseThrottled = "jwt.token.reuse.throttled"
	EventTokenReuseMalicious = "jwt.token.reuse.malicious"
)

var (
	ErrRefreshTokenIsRequired = errors.New("refresh token is required")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
)
