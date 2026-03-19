package constants

const (
	EventTOTPEnabled       = "totp.enabled"
	EventTOTPDisabled      = "totp.disabled"
	EventTOTPVerified      = "totp.verified"
	EventTOTPBackupUsed    = "totp.backup_code_used"
	EventTOTPDeviceTrusted = "totp.device_trusted"
)

const (
	CookieTOTPPending = "totp_pending"
	CookieTOTPTrusted = "totp_trusted"
)
