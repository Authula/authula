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
	ProviderID string `path:"provider"`
	Code       string `query:"code" json:"code,omitempty" nullable:"true"`
	State      string `query:"state" json:"state,omitempty" nullable:"true"`
	Error      string `query:"error" json:"error,omitempty" nullable:"true"`
	// User carries the form-posted user payload that some providers send on
	// first consent only, outside the token.
	User string `json:"user,omitempty" nullable:"true"`

	// ActorID and SessionID are populated from the request context when a signed-in
	// user starts a linking flow.
	ActorID   string `json:"-"`
	SessionID string `json:"-"`

	// Signed flow cookies are validated by the use case.
	StateCookie    string `json:"-"`
	RedirectCookie string `json:"-"`
	VerifierCookie string `json:"-"`
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
