package constants

import "errors"

var (
	ErrProviderNotFound             = errors.New("oauth provider not found")
	ErrAccessDenied                 = errors.New("oauth provider denied the request")
	ErrInvalidState                 = errors.New("invalid or expired oauth state")
	ErrStateProviderMismatch        = errors.New("oauth state does not belong to this provider")
	ErrInvalidCode                  = errors.New("missing oauth authorization code")
	ErrInvalidCallback              = errors.New("invalid oauth callback request")
	ErrPKCEVerifierMissing          = errors.New("missing oauth PKCE verifier")
	ErrPKCEChallengeMissing         = errors.New("oauth authorization URL is missing a S256 PKCE challenge")
	ErrEmailRequired                = errors.New("oauth provider did not return an email address")
	ErrEmailNotVerified             = errors.New("oauth provider email is not verified")
	ErrProviderAccountIDRequired    = errors.New("oauth provider did not return an account id")
	ErrAccountConflict              = errors.New("oauth account conflicts with an existing account")
	ErrAccountLinkedToAnotherUser   = errors.New("oauth account is linked to another user")
	ErrEmailInUse                   = errors.New("email already in use")
	ErrTokenExpired                 = errors.New("oauth token is expired")
	ErrUnauthorized                 = errors.New("authentication required")
	ErrProviderFailure              = errors.New("oauth provider request failed")
	ErrInvalidRedirectTo            = errors.New("invalid redirect_to")
	ErrSessionNotFound              = errors.New("session not found")
	ErrOAuthSecretNotConfigured     = errors.New("oauth2 requires a non-empty application secret")
	ErrProviderConfigInvalid        = errors.New("invalid oauth provider configuration")
	ErrProviderConfigurationPartial = errors.New("partial oauth provider credentials")
)
