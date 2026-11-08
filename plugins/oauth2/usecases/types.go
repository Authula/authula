package usecases

import "fmt"

// AuthorizeResult contains the result of an authorization request.
type AuthorizeResult struct {
	AuthorizationURL string
	Binding          string
}

// RedirectError asks the HTTP layer to redirect the browser instead of returning
// a JSON error. It is used for provider error callbacks when a trusted
// redirect_to was stored with the flow state.
type RedirectError struct {
	RedirectTo  string
	Code        string
	Description string
}

func (e *RedirectError) Error() string {
	if e.Description == "" {
		return fmt.Sprintf("oauth redirect error: %s", e.Code)
	}
	return fmt.Sprintf("oauth redirect error: %s: %s", e.Code, e.Description)
}
