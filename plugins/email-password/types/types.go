package types

import (
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type EmailPasswordPluginConfig struct {
	Enabled                  bool          `json:"enabled" toml:"enabled"`
	MinPasswordLength        int           `json:"min_password_length" toml:"min_password_length"`
	MaxPasswordLength        int           `json:"max_password_length" toml:"max_password_length"`
	DisableSignUp            bool          `json:"disable_sign_up" toml:"disable_sign_up"`
	RequireEmailVerification bool          `json:"require_email_verification" toml:"require_email_verification"`
	AutoSignIn               bool          `json:"auto_sign_in" toml:"auto_sign_in"`
	SendEmailOnSignUp        bool          `json:"send_email_on_sign_up" toml:"send_email_on_sign_up"`
	SendEmailOnSignIn        bool          `json:"send_email_on_sign_in" toml:"send_email_on_sign_in"`
	ExpiresIn                time.Duration `json:"expires_in" toml:"expires_in"`
	// Dynamic function to send email verification emails in library mode
	SendEmailVerification func(params SendEmailVerificationParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
}

type SendEmailVerificationParams struct {
	User  models.User
	URL   string
	Token string
}

type SignUpResult struct {
	User *models.User
}

type SignInResult struct {
	User *models.User
}
