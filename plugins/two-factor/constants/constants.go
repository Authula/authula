package constants

import "errors"

const (
	EventTwoFactorEnabled       = "two_factor.enabled"
	EventTwoFactorDisabled      = "two_factor.disabled"
	EventTwoFactorVerified      = "two_factor.verified"
	EventTwoFactorBackupUsed    = "two_factor.backup_code_used"
	EventTwoFactorDeviceTrusted = "two_factor.device_trusted"
)

var (
	ErrTwoFactorNotEnabled     = errors.New("two-factor authentication is not enabled")
	ErrTwoFactorAlreadyEnabled = errors.New("two-factor authentication is already enabled")
	ErrInvalidTOTPCode         = errors.New("invalid TOTP code")
	ErrInvalidBackupCode       = errors.New("invalid backup code")
	ErrInvalidPendingToken     = errors.New("invalid or expired pending token")
	ErrPasswordRequired        = errors.New("password is required")
	ErrInvalidPassword         = errors.New("invalid password")
	ErrAccountNotFound         = errors.New("credential account not found")
	ErrUserNotFound            = errors.New("user not found")
)
