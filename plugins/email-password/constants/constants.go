package constants

import "errors"

const (
	EventUserSignedUp        = "user.signed_up"
	EventUserSignedIn        = "user.signed_in"
	EventUserEmailVerified   = "user.verified_email"
	EventUserChangedPassword = "user.changed_password"
	EventUserEmailChanged    = "user.changed_email"
)

var (
	ErrInvalidPasswordLength = errors.New("password length invalid")
	ErrAccountNotFound       = errors.New("account not found")
	ErrInvalidOrExpiredToken = errors.New("invalid or expired token")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserNotAuthorized     = errors.New("you are not authorized to perform this action")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrEmailNotVerified      = errors.New("email not verified")
	ErrSignUpDisabled        = errors.New("sign up is disabled")
	ErrEmailAlreadyExists    = errors.New("email already registered")
	ErrPasswordLengthInvalid = errors.New("password length invalid")
)
