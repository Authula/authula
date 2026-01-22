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
	ErrInvalidPasswordLength        = errors.New("password length invalid")
	ErrEmailAlreadyExists           = errors.New("email already registered")
	ErrInvalidCredentials           = errors.New("invalid credentials")
	ErrUserNotFound                 = errors.New("user not found")
	ErrAccountNotFound              = errors.New("account not found")
	ErrEmailNotVerified             = errors.New("email not verified")
	ErrSignUpDisabled               = errors.New("sign up is disabled")
	ErrInvalidOrExpiredToken        = errors.New("invalid or expired token")
	ErrUserNotAuthorized            = errors.New("you are not authorized to perform this action")
	ErrInvalidEmailVerificationType = errors.New("invalid email verification type")
	ErrInvalidEmailMatch            = errors.New("invalid email match")
)
