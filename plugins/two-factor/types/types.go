package types

import (
	"net/http"
	"strings"
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
	TrustedDeviceDuration    time.Duration `json:"trusted_device_duration" toml:"trusted_device_duration"`
	PendingTokenExpiry       time.Duration `json:"pending_token_expiry" toml:"pending_token_expiry"`
	SecureCookie             bool          `json:"secure_cookie" toml:"secure_cookie"`
	SameSite                 string        `json:"same_site" toml:"same_site"`
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
	if c.TrustedDeviceDuration == 0 {
		c.TrustedDeviceDuration = 30 * 24 * time.Hour
	}
	if c.PendingTokenExpiry == 0 {
		c.PendingTokenExpiry = 5 * time.Minute
	}
	if c.SameSite == "" {
		c.SameSite = "lax"
	}
}

// ParseSameSite converts a string to http.SameSite.
// Accepted values: "strict", "lax", "none" (case-insensitive).
// Defaults to http.SameSiteLaxMode for unrecognized values.
func ParseSameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
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

type ViewBackupCodesRequest struct {
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
	RemainingCount int `json:"remainingCount"`
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
	User                  *models.User
	Session               *models.Session
	SessionToken          string
	TrustedDeviceToken    string        // empty if not trusting
	TrustedDeviceDuration time.Duration // for cookie MaxAge
	SecureCookie          bool
	SameSite              http.SameSite
}
