package types

import (
	"time"

	"github.com/Authula/authula/models"
)

// AuthorizeRequest represents an authorization request
type AuthorizeRequest struct {
	ProviderID string `json:"provider" required:"true" nullable:"false"`
	RedirectTo string `json:"redirect_to,omitempty" nullable:"true"`
}

// AuthorizeResponse represents an authorization response
type AuthorizeResponse struct {
	AuthURL string `json:"auth_url" required:"true" nullable:"false"`
}

// CallbackRequest represents an OAuth2 callback request
type CallbackRequest struct {
	ProviderID string `json:"provider" required:"true" nullable:"false"`
	Code       string `json:"code" required:"true" nullable:"false"`
	State      string `json:"state" required:"true" nullable:"false"`
	Error      string `json:"error,omitempty" nullable:"true"`
}

// CallbackResult represents the result of OAuth2 callback
type CallbackResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

// CallbackResponse represents an OAuth2 callback response
type CallbackResponse struct {
	User    *models.User    `json:"user" required:"true" nullable:"false"`
	Session *models.Session `json:"session" required:"true" nullable:"false"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	Provider string
	UserID   string
}

// RefreshResponse represents a token refresh response
type RefreshResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// LinkAccountRequest represents an account linking request
type LinkAccountRequest struct {
	Provider   string
	UserID     string
	RedirectTo string
}

// LinkAccountResponse represents an account linking response
type LinkAccountResponse struct {
	User *models.User `json:"user,omitempty"`
}
