package types

import (
	"net/http"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type TOTPPluginConfig struct {
	Enabled                       bool          `json:"enabled" toml:"enabled"`
	SkipVerificationOnEnable      bool          `json:"skip_verification_on_enable" toml:"skip_verification_on_enable"`
	BackupCodeCount               int           `json:"backup_code_count" toml:"backup_code_count"`
	TrustedDeviceDuration         time.Duration `json:"trusted_device_duration" toml:"trusted_device_duration"`
	TrustedDevicesAutoCleanup     bool          `json:"trusted_devices_auto_cleanup" toml:"trusted_devices_auto_cleanup"`
	TrustedDevicesCleanupInterval time.Duration `json:"trusted_devices_cleanup_interval" toml:"trusted_devices_cleanup_interval"`
	PendingTokenExpiry            time.Duration `json:"pending_token_expiry" toml:"pending_token_expiry"`
	SecureCookie                  bool          `json:"secure_cookie" toml:"secure_cookie"`
	SameSite                      string        `json:"same_site" toml:"same_site"`
}

func (c *TOTPPluginConfig) ApplyDefaults() {
	if c.BackupCodeCount == 0 {
		c.BackupCodeCount = 10
	}
	if c.TrustedDeviceDuration == 0 {
		c.TrustedDeviceDuration = 30 * 24 * time.Hour
	}
	if c.TrustedDevicesAutoCleanup && c.TrustedDevicesCleanupInterval == 0 {
		c.TrustedDevicesCleanupInterval = time.Hour
	}
	if c.PendingTokenExpiry == 0 {
		c.PendingTokenExpiry = 5 * time.Minute
	}
	if c.SameSite == "" {
		c.SameSite = "lax"
	}
}

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

type VerifyTOTPRequest struct {
	Code        string `json:"code"`
	TrustDevice bool   `json:"trust_device,omitempty"`
}

type VerifyBackupCodeRequest struct {
	Code        string `json:"code"`
	TrustDevice bool   `json:"trust_device,omitempty"`
}

// Response payloads
type EnableResponse struct {
	TotpURI     string   `json:"totp_uri"`
	BackupCodes []string `json:"backup_codes"`
}

type GetTOTPURIResponse struct {
	TotpURI string `json:"totp_uri"`
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
	BackupCodes []string `json:"backup_codes"`
}

type ViewBackupCodesResponse struct {
	RemainingCount int `json:"remaining_count"`
}

type TOTPRedirectResponse struct {
	TOTPRedirect bool `json:"totp_redirect"`
}

// Internal result types
type EnableResult struct {
	TotpURI      string
	BackupCodes  []string
	PendingToken string
}

type VerifyResult struct {
	User               *models.User
	Session            *models.Session
	SessionToken       string
	TrustedDeviceToken string // empty if not issued
}
