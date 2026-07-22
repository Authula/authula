package types

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
)

type EmailPasswordPluginConfig struct {
	Enabled                     bool          `json:"enabled" toml:"enabled"`
	MinPasswordLength           int           `json:"min_password_length" toml:"min_password_length"`
	MaxPasswordLength           int           `json:"max_password_length" toml:"max_password_length"`
	DisableSignUp               bool          `json:"disable_sign_up" toml:"disable_sign_up"`
	RequireEmailVerification    bool          `json:"require_email_verification" toml:"require_email_verification"`
	AutoSignIn                  bool          `json:"auto_sign_in" toml:"auto_sign_in"`
	SendEmailOnSignUp           bool          `json:"send_email_on_sign_up" toml:"send_email_on_sign_up"`
	SendEmailOnSignIn           bool          `json:"send_email_on_sign_in" toml:"send_email_on_sign_in"`
	EmailVerificationExpiresIn  time.Duration `json:"email_verification_expires_in" toml:"email_verification_expires_in"`
	PasswordResetExpiresIn      time.Duration `json:"password_reset_expires_in" toml:"password_reset_expires_in"`
	RequestEmailChangeExpiresIn time.Duration `json:"request_email_change_expires_in" toml:"request_email_change_expires_in"`

	SendEmailVerification       func(params SendEmailVerificationParams, reqCtx *models.RequestContext) error       `json:"-" toml:"-"`
	SendPasswordResetEmail      func(params SendPasswordResetEmailParams, reqCtx *models.RequestContext) error      `json:"-" toml:"-"`
	SendChangedPasswordEmail    func(params SendChangedPasswordEmailParams, reqCtx *models.RequestContext) error    `json:"-" toml:"-"`
	SendRequestEmailChangeEmail func(params SendRequestEmailChangeEmailParams, reqCtx *models.RequestContext) error `json:"-" toml:"-"`
	SendChangedEmailToOldEmail  func(params SendChangedEmailToOldEmailParams, reqCtx *models.RequestContext) error  `json:"-" toml:"-"`
	SendChangedEmailToNewEmail  func(params SendChangedEmailToNewEmailParams, reqCtx *models.RequestContext) error  `json:"-" toml:"-"`

	ServiceHooks *EmailPasswordServiceHooksConfig `json:"-" toml:"-"`
}

func (config *EmailPasswordPluginConfig) ApplyDefaults() {
	if config.MinPasswordLength == 0 {
		config.MinPasswordLength = 8
	}
	if config.MaxPasswordLength == 0 {
		config.MaxPasswordLength = 128
	}
	if config.EmailVerificationExpiresIn == 0 {
		config.EmailVerificationExpiresIn = 24 * time.Hour
	}
	if config.PasswordResetExpiresIn == 0 {
		config.PasswordResetExpiresIn = time.Hour
	}
	if config.RequestEmailChangeExpiresIn == 0 {
		config.RequestEmailChangeExpiresIn = time.Hour
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

type SendChangedPasswordEmailParams struct {
	User models.User
}

type SendEmailResetEmailParams struct {
	User  models.User
	URL   string
	Token string
}

type SendRequestEmailChangeEmailParams struct {
	User     models.User
	URL      string
	Token    string
	NewEmail string
	OldEmail string
}

type SendChangedEmailToOldEmailParams struct {
	User  models.User
	Email string
}

type SendChangedEmailToNewEmailParams struct {
	User  models.User
	Email string
}

type EmailPasswordServiceHooksConfig struct {
	SignUp            *SignUpServiceHooksConfig
	SignIn            *SignInServiceHooksConfig
	EmailVerification *EmailVerificationServiceHooksConfig
	PasswordReset     *PasswordResetServiceHooksConfig
	PasswordChange    *PasswordChangeServiceHooksConfig
	EmailChange       *EmailChangeServiceHooksConfig
}

type SignUpServiceHooksConfig struct {
	BeforeSignUp func(ctx context.Context, user *models.User) error
	AfterSignUp  func(ctx context.Context, result *SignUpResult) error
}

type SignInServiceHooksConfig struct {
	BeforeSignIn func(ctx context.Context, user *models.User) error
	AfterSignIn  func(ctx context.Context, result *SignInResult) error
}

type EmailVerificationServiceHooksConfig struct {
	AfterVerifyEmail func(ctx context.Context, user *models.User, verificationType models.VerificationType) error
}

type PasswordResetServiceHooksConfig struct {
	BeforeRequestPasswordReset func(ctx context.Context, user *models.User) error
}

type PasswordChangeServiceHooksConfig struct {
	BeforeChangePassword func(ctx context.Context, user *models.User, newPassword string) error
	AfterChangePassword  func(ctx context.Context, user *models.User) error
}

type EmailChangeServiceHooksConfig struct {
	BeforeRequestEmailChange func(ctx context.Context, user *models.User) error
	AfterEmailChanged        func(ctx context.Context, user *models.User, oldEmail, newEmail string) error
}
