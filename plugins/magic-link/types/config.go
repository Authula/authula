package types

import (
	"time"

	"github.com/Authula/authula/models"
)

type MagicLinkPluginConfig struct {
	Enabled       bool          `json:"enabled" toml:"enabled"`
	ExpiresIn     time.Duration `json:"expires_in" toml:"expires_in"`
	DisableSignUp bool          `json:"disable_sign_up" toml:"disable_sign_up"`

	SendMagicLinkVerificationEmail func(params SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
}

func (config *MagicLinkPluginConfig) ApplyDefaults() {
	if config.ExpiresIn <= 0 {
		config.ExpiresIn = 15 * time.Minute
	}
}

type SendMagicLinkVerificationEmailParams struct {
	Email string
	URL   string
	Token string
}
