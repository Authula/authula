package types

import (
	"time"

	"github.com/Authula/authula/models"
)

// AuthorizeRequest represents an authorization request
type AuthorizeRequest struct {
	ProviderID string `path:"provider"`
	RedirectTo string `query:"redirect_to" json:"redirect_to,omitempty" nullable:"true"`
	// ActorID is set when an already signed-in user starts a linking flow.
	ActorID string `json:"-"`
}

// AuthorizeResponse represents an authorization response
type AuthorizeResponse struct {
	AuthURL string `json:"auth_url" required:"true" nullable:"false"`
}

// CallbackRequest represents an OAuth2 callback request
type CallbackRequest struct {
	ProviderID       string `path:"provider"`
	Code             string `query:"code" json:"code,omitempty" nullable:"true"`
	State            string `query:"state" json:"state,omitempty" nullable:"true"`
	Error            string `query:"error" json:"error,omitempty" nullable:"true"`
	ErrorDescription string `query:"error_description" json:"error_description,omitempty" nullable:"true"`

	// ActorID and SessionID are populated from the request context when a signed-in
	// user starts a linking flow.
	ActorID   string `json:"-"`
	SessionID string `json:"-"`

	// Binding is populated from the browser-binding cookie and is folded into the
	// state lookup hash.
	Binding string `json:"-"`
}

// CallbackResult represents the result of OAuth2 callback
type CallbackResult struct {
	User          *models.User
	Session       *models.Session
	SessionToken  string
	RedirectTo    string
	ReusedSession bool
}

// CallbackResponse represents an OAuth2 callback response.
// It intentionally omits the session token, IP address, and user agent.
type CallbackResponse struct {
	User    *models.User     `json:"user" required:"true" nullable:"false"`
	Session *CallbackSession `json:"session" required:"true" nullable:"false"`
}

// CallbackSession is the safe public view of a session.
type CallbackSession struct {
	ID        string    `json:"id" required:"true" nullable:"false"`
	UserID    string    `json:"user_id" required:"true" nullable:"false"`
	ExpiresAt time.Time `json:"expires_at" required:"true" nullable:"false"`
}

// ErrorResponse is the generic error body returned by OAuth2 endpoints.
type ErrorResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

// RedirectResponse documents the empty body returned with a 302 response.
type RedirectResponse struct{}

// TokenRequest is the use-case input for a client credentials token request.
// Scope is a pointer so an omitted scope is distinguishable from scope=.
type TokenRequest struct {
	ClientID     string
	ClientSecret string
	Scope        *string
}

// TokenResult is the use-case result for a client credentials token request.
type TokenResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   time.Duration
	Scopes      []string
}

// TokenResponse is the successful token endpoint response.
type TokenResponse struct {
	AccessToken string `json:"access_token" required:"true" nullable:"false"`
	TokenType   string `json:"token_type" required:"true" nullable:"false"`
	ExpiresIn   int64  `json:"expires_in" required:"true" nullable:"false"`
	Scope       string `json:"scope,omitempty"`
}

// TokenErrorResponse is the OAuth 2.0 error body returned with a 4xx response.
type TokenErrorResponse struct {
	Error            string `json:"error" required:"true" nullable:"false"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// TokenEndpointRequest documents the token endpoint form body.
type TokenEndpointRequest struct {
	GrantType    string `json:"grant_type" formData:"grant_type"`
	ClientID     string `json:"client_id" formData:"client_id"`
	ClientSecret string `json:"client_secret" formData:"client_secret"`
	Scope        string `json:"scope" formData:"scope"`
}
