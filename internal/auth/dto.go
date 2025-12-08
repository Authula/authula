package auth

import (
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// SignInResult represents the result of a sign-in operation
type SignInResult struct {
	Token     string       `json:"token"`
	User      *domain.User `json:"user"`
	CSRFToken string       `json:"csrf_token,omitempty"`
}

// SignUpResult represents the result of a sign-up operation
type SignUpResult struct {
	Token     string       `json:"token,omitempty"`
	User      *domain.User `json:"user"`
	CSRFToken string       `json:"csrf_token,omitempty"`
}

// SignOutResult represents the result of a sign-out operation
type SignOutResult struct {
	Message string `json:"message"`
}

// VerifyEmailResult represents the result of email verification
type VerifyEmailResult struct {
	Message string       `json:"message"`
	User    *domain.User `json:"user,omitempty"`
}

// PasswordResetRequestResult represents the result of a password reset request
type PasswordResetRequestResult struct {
	Message string `json:"message"`
}

// PasswordResetResult represents the result of a password reset
type PasswordResetResult struct {
	Message string `json:"message"`
}

// EmailChangeRequestResult represents the result of an email change request
type EmailChangeRequestResult struct {
	Message string `json:"message"`
}

// EmailChangeResult represents the result of confirming an email change
type EmailChangeResult struct {
	Message string       `json:"message"`
	User    *domain.User `json:"user,omitempty"`
}
