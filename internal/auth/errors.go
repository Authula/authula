package auth

import "errors"

var (
	// Authentication errors
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrPasswordHashingFailed = errors.New("password hashing failed")

	// Token errors
	ErrTokenGenerationFailed = errors.New("token generation failed")
	ErrTokenExpired          = errors.New("token expired")
	ErrInvalidToken          = errors.New("invalid token")
	ErrNoRefreshToken        = errors.New("no refresh token available")

	// Session errors
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionCreationFailed = errors.New("session creation failed")
	ErrSessionDeletionFailed = errors.New("session deletion failed")

	// Verification errors
	ErrVerificationNotFound = errors.New("verification token not found")
	ErrVerificationExpired  = errors.New("verification token expired")
	ErrVerificationInvalid  = errors.New("invalid verification token")

	// Account errors
	ErrAccountNotFound        = errors.New("account not found")
	ErrAccountCreationFailed  = errors.New("account creation failed")
	ErrAccountUpdateFailed    = errors.New("account update failed")
	ErrAccountLinkingRequired = errors.New("account linking required: user exists with a different provider")

	// Email verification errors
	ErrEmailVerificationFailed = errors.New("email verification failed")

	// Email change errors
	ErrEmailAlreadyExists       = errors.New("email already exists")
	ErrEmailChangeRequestFailed = errors.New("email change request failed")

	// Password reset errors
	ErrPasswordResetFailed        = errors.New("password reset failed")
	ErrPasswordResetRequestFailed = errors.New("password reset request failed")

	// Configuration errors
	ErrConfigInvalid = errors.New("invalid configuration")

	// OAuth2 errors
	ErrOAuth2ProviderNotConfigured = errors.New("oauth2 provider not configured")
	ErrOAuth2ExchangeFailed        = errors.New("oauth2 token exchange failed")
	ErrOAuth2UserInfoFailed        = errors.New("failed to get oauth2 user info")
)
