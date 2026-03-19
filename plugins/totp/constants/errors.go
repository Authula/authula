package constants

import "errors"

var (
	ErrTOTPNotEnabled          = errors.New("totp authentication is not enabled")
	ErrTOTPAlreadyEnabled      = errors.New("totp authentication is already enabled")
	ErrInvalidTOTPCode         = errors.New("invalid totp code")
	ErrInvalidBackupCode       = errors.New("invalid backup code")
	ErrInvalidPendingToken     = errors.New("invalid or expired pending token")
	ErrPasswordRequired        = errors.New("password is required")
	ErrInvalidPassword         = errors.New("invalid password")
	ErrAccountNotFound         = errors.New("credential account not found")
	ErrUserNotFound            = errors.New("user not found")
	ErrInvalidVerificationType = errors.New("invalid verification type")
	ErrPendingTokenExpired     = errors.New("pending token has expired")
)
