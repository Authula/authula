package models

type OAuth2UserInfo struct {
	ID       string
	Email    string
	Name     string
	Picture  string
	Verified bool
}

// OAuth2LoginResult contains the information needed for the OAuth2 login flow
type OAuth2LoginResult struct {
	AuthURL  string  // The authorization URL to redirect to
	State    string  // CSRF protection state
	Verifier *string // PKCE code verifier (if PKCE is required)
}
