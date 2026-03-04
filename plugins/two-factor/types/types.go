package types

import (
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type TwoFactorPluginConfig struct {
	Enabled                  bool          `json:"enabled" toml:"enabled"`
	Issuer                   string        `json:"issuer" toml:"issuer"`
	Digits                   int           `json:"digits" toml:"digits"`
	Period                   int           `json:"period" toml:"period"`
	SkipVerificationOnEnable bool          `json:"skip_verification_on_enable" toml:"skip_verification_on_enable"`
	BackupCodeCount          int           `json:"backup_code_count" toml:"backup_code_count"`
	BackupCodeLength         int           `json:"backup_code_length" toml:"backup_code_length"`
	TrustedDeviceDuration    time.Duration `json:"trusted_device_duration" toml:"trusted_device_duration"`
	PendingTokenExpiry       time.Duration `json:"pending_token_expiry" toml:"pending_token_expiry"`
}

func (c *TwoFactorPluginConfig) ApplyDefaults() {
	if c.Digits == 0 {
		c.Digits = 6
	}
	if c.Period == 0 {
		c.Period = 30
	}
	if c.BackupCodeCount == 0 {
		c.BackupCodeCount = 10
	}
	if c.BackupCodeLength == 0 {
		c.BackupCodeLength = 10
	}
	if c.TrustedDeviceDuration == 0 {
		c.TrustedDeviceDuration = 30 * 24 * time.Hour
	}
	if c.PendingTokenExpiry == 0 {
		c.PendingTokenExpiry = 5 * time.Minute
	}
}

// Request payloads
type EnableRequest struct {
	Password string `json:"password"`
	Issuer   string `json:"issuer,omitempty"`
}

type DisableRequest struct {
	Password string `json:"password"`
}

type GetTOTPURIRequest struct {
	Password string `json:"password"`
}

type VerifyTOTPRequest struct {
	Code        string `json:"code"`
	TrustDevice bool   `json:"trustDevice,omitempty"`
}

type VerifyBackupCodeRequest struct {
	Code        string `json:"code"`
	TrustDevice bool   `json:"trustDevice,omitempty"`
}

type GenerateBackupCodesRequest struct {
	Password string `json:"password"`
}

// Response payloads
type EnableResponse struct {
	TotpURI     string   `json:"totpURI"`
	BackupCodes []string `json:"backupCodes"`
}

type GetTOTPURIResponse struct {
	TotpURI string `json:"totpURI"`
}

type VerifyTOTPResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type VerifyBackupCodeResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type GenerateBackupCodesResponse struct {
	BackupCodes []string `json:"backupCodes"`
}

type ViewBackupCodesResponse struct {
	BackupCodes []string `json:"backupCodes"`
}

type TwoFactorRedirectResponse struct {
	TwoFactorRedirect bool `json:"twoFactorRedirect"`
}

// Internal result types
type EnableResult struct {
	TotpURI     string
	BackupCodes []string
}

type VerifyResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}
