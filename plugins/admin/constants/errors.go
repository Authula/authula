package constants

import "errors"

var (
	ErrUserIDRequired                  = errors.New("user ID is required")
	ErrProviderIDRequired              = errors.New("provider ID is required")
	ErrAccountIDRequired               = errors.New("account ID is required")
	ErrAccessTokenExpiresAtBeforeNow   = errors.New("access token expiration time must be in the future")
	ErrRefreshTokenExpiresAtBeforeNow  = errors.New("refresh token expiration time must be in the future")
	ErrNoPropertiesProvided            = errors.New("at least one property must be provided for update")
	ErrBannedUntilBeforeNow            = errors.New("banned until time must be in the future")
	ErrImpersonationExpiresAtBeforeNow = errors.New("impersonation expiration time must be in the future")
)
