package types

import (
	"net/http"
	"strings"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/totp/constants"
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

type DisableResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type VerifyTOTPRequest struct {
	Code        string `json:"code" required:"true" nullable:"false"`
	TrustDevice bool   `json:"trust_device,omitempty" nullable:"false"`
}

func (v *VerifyTOTPRequest) Validate() error {
	if v.Code == "" {
		return constants.ErrBackupCodeIsRequired
	}
	return nil
}

type VerifyBackupCodeRequest struct {
	Code        string `json:"code" required:"true" nullable:"false"`
	TrustDevice bool   `json:"trust_device,omitempty" nullable:"false"`
}

func (v *VerifyBackupCodeRequest) Validate() error {
	if v.Code == "" {
		return constants.ErrBackupCodeIsRequired
	}
	return nil
}

// Response payloads
type EnableResponse struct {
	TotpURI     string   `json:"totp_uri" required:"true" nullable:"false"`
	BackupCodes []string `json:"backup_codes" required:"true" nullable:"false"`
}

type GetTOTPURIResponse struct {
	TotpURI string `json:"totp_uri" required:"true" nullable:"false"`
}

type VerifyTOTPResponse struct {
	User    *models.User    `json:"user" required:"true" nullable:"false"`
	Session *models.Session `json:"session" required:"true" nullable:"false"`
}

type VerifyBackupCodeResponse struct {
	User    *models.User    `json:"user" required:"true" nullable:"false"`
	Session *models.Session `json:"session" required:"true" nullable:"false"`
}

type GenerateBackupCodesResponse struct {
	BackupCodes []string `json:"backup_codes" required:"true" nullable:"false"`
}

type ViewBackupCodesResponse struct {
	RemainingCount int `json:"remaining_count" required:"true" nullable:"false"`
}

type TOTPRedirectResponse struct {
	TOTPRedirect bool `json:"totp_redirect" required:"true" nullable:"false"`
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
