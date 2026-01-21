package types

import (
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type EmailPasswordPluginConfig struct {
	Enabled                  bool `json:"enabled" toml:"enabled"`
	MinPasswordLength        int  `json:"min_password_length" toml:"min_password_length"`
	MaxPasswordLength        int  `json:"max_password_length" toml:"max_password_length"`
	DisableSignUp            bool `json:"disable_sign_up" toml:"disable_sign_up"`
	RequireEmailVerification bool `json:"require_email_verification" toml:"require_email_verification"`
	// Whether to automatically sign in the user after sign-up
	AutoSignIn                 bool          `json:"auto_sign_in" toml:"auto_sign_in"`
	SendEmailOnSignUp          bool          `json:"send_email_on_sign_up" toml:"send_email_on_sign_up"`
	SendEmailOnSignIn          bool          `json:"send_email_on_sign_in" toml:"send_email_on_sign_in"`
	EmailVerificationExpiresIn time.Duration `json:"email_verification_expires_in" toml:"email_verification_expires_in"`
	PasswordResetExpiresIn     time.Duration `json:"password_reset_expires_in" toml:"password_reset_expires_in"`
	EmailResetExpiresIn        time.Duration `json:"email_reset_expires_in" toml:"email_reset_expires_in"`
	// Function to override sending email verification emails
	SendEmailVerification func(params SendEmailVerificationParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
	// Function to override sending password reset emails
	SendPasswordResetEmail func(params SendPasswordResetEmailParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
	// Function to override sending email reset emails
	SendEmailResetEmail func(params SendEmailResetEmailParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
	// Function to override sending email change emails
	SendEmailChangeEmail func(params SendEmailChangeEmailParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
}

func (cfg *EmailPasswordPluginConfig) ApplyDefaults() {
	if cfg.MinPasswordLength == 0 {
		cfg.MinPasswordLength = 8
	}
	if cfg.MaxPasswordLength == 0 {
		cfg.MaxPasswordLength = 128
	}
	if cfg.EmailVerificationExpiresIn == 0 {
		cfg.EmailVerificationExpiresIn = 24 * time.Hour
	}
	if cfg.PasswordResetExpiresIn == 0 {
		cfg.PasswordResetExpiresIn = time.Hour
	}
	if cfg.EmailResetExpiresIn == 0 {
		cfg.EmailResetExpiresIn = time.Hour
	}
}

type SendEmailVerificationParams struct {
	User  models.User
	URL   string
	Token string
}

type SendPasswordResetEmailParams struct {
	User  models.User
	URL   string
	Token string
}

type SendEmailResetEmailParams struct {
	User  models.User
	URL   string
	Token string
}

type SendEmailChangeEmailParams struct {
	User     models.User
	URL      string
	Token    string
	NewEmail string
	OldEmail string
}

type SignUpResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken *string // Nil if auto sign-in is disabled
}

type SignUpResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type SignInResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

type SignInResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type ChangePasswordResponse struct {
	Message string `json:"message"`
}

type ChangeEmailResponse struct {
	Message string `json:"message"`
}
