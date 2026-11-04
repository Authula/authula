package usecases

// AuthorizeResult contains the result of an authorization request
type AuthorizeResult struct {
	AuthorizationURL string
	StateCookie      string
	VerifierCookie   *string
	RedirectCookie   string
}
