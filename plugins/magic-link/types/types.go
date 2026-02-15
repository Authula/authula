package types

import (
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type MagicLinkPluginConfig struct {
	Enabled       bool          `json:"enabled" toml:"enabled"`
	ExpiresIn     time.Duration `json:"expires_in" toml:"expires_in"`
	DisableSignUp bool          `json:"disable_sign_up" toml:"disable_sign_up"`
	// Custom function to override sending the magic link verification email
	SendMagicLinkVerificationEmail func(email string, url string, token string) error `json:"-" toml:"-"`
}

func (config *MagicLinkPluginConfig) ApplyDefaults() {
	if config.ExpiresIn == 0 {
		config.ExpiresIn = 15 * time.Minute
	}
}

type SignInResult struct {
	Token string
}

type SignInResponse struct {
	Message string `json:"message"`
}

type VerifyResponse struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

type ExchangeRequest struct {
	Token string `json:"token"`
}

type ExchangeResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

type ExchangeResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}
