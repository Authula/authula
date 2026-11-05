package constants

import "time"

const (
	CookieState      = "oauth2_state"
	CookieVerifier   = "oauth2_verifier"
	CookieRedirectTo = "oauth2_redirect_to"

	// FlowCookieTTL is the lifetime of the temporary OAuth2 flow cookies.
	FlowCookieTTL = 5 * time.Minute
)
