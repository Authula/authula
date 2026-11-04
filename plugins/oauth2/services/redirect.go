package services

import (
	"fmt"

	"github.com/Authula/authula/util"
)

// ValidateRedirectTo validates a redirect destination and returns its normalised form.
// Relative paths and absolute URLs on trusted origins are allowed.
func ValidateRedirectTo(redirectTo string, trustedOrigins []string) (string, error) {
	if redirectTo == "" {
		return "/", nil
	}

	parsed, err := util.IsTrustedCallbackURL(redirectTo, trustedOrigins)
	if err != nil {
		return "", fmt.Errorf("invalid redirect_to: %w", err)
	}

	return parsed.String(), nil
}
