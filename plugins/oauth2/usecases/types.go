package usecases

import (
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

// AuthorizeResult contains the result of an authorization request
type AuthorizeResult struct {
	AuthorizationURL string
	StateCookie      string
	VerifierCookie   *string
	RedirectCookie   string
}

// CallbackResult contains the result of a callback
type CallbackResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
	RedirectTo   string
}

// RefreshResult contains the result of a token refresh
type RefreshResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
}

// LinkAccountResult contains the result of linking an account
type LinkAccountResult struct {
	ProviderID string
	AccountID  string
}
