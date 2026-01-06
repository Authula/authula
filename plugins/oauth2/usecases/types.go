package usecases

import (
	"github.com/GoBetterAuth/go-better-auth/models"
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
	Error        string
	Success      bool
}

// RefreshResult contains the result of a token refresh
type RefreshResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
	Error       string
	Success     bool
}

// LinkAccountResult contains the result of linking an account
type LinkAccountResult struct {
	Error      string
	Success    bool
	ProviderID string
	AccountID  string
	LinkedAt   string
}
