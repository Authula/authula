package types

import (
	"time"

	"github.com/Authula/authula/models"
)

// AuthorizeRequest represents an authorization request
type AuthorizeRequest struct {
	ProviderID string
	RedirectTo string
}

// AuthorizeResponse represents an authorization response
type AuthorizeResponse struct {
	AuthURL string `json:"auth_url"`
}

// CallbackRequest represents an OAuth2 callback request
type CallbackRequest struct {
	ProviderID string
	Code       string
	State      string
	Error      string
}

// CallbackResult represents the result of OAuth2 callback
type CallbackResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

// CallbackResponse represents an OAuth2 callback response
type CallbackResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
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
