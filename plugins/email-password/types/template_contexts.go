package types

import (
	"time"

	emailtmpl "github.com/Authula/authula/internal/email/template"
)

type VerifyEmailContext struct {
	emailtmpl.CommonContext
	UserEmail        string
	VerificationLink string
	Expiry           time.Duration
}

type PasswordResetContext struct {
	emailtmpl.CommonContext
	UserEmail string
	ResetLink string
	Expiry    time.Duration
}

type PasswordChangedContext struct {
	emailtmpl.CommonContext
	UserEmail string
}

type EmailChangeRequestContext struct {
	emailtmpl.CommonContext
	UserEmail  string
	OldEmail   string
	NewEmail   string
	ChangeLink string
	Expiry     time.Duration
}

type EmailChangedNotificationContext struct {
	emailtmpl.CommonContext
	UserEmail string
	OldEmail  string
	NewEmail  string
}
