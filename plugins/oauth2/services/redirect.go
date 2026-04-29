package services

import (
	"fmt"
	"net/url"

	"github.com/Authula/authula/internal/util"
)

// ValidateRedirectTo validates a redirect URL against trusted origins
func ValidateRedirectTo(redirectTo string, trustedOrigins []string) error {
	if redirectTo == "" {
		return fmt.Errorf("redirect_to is required")
	}

	// Parse the redirect URL
	u, err := url.Parse(redirectTo)
	if err != nil {
		return fmt.Errorf("invalid redirect_to URL: %w", err)
	}

	// Redirect must be absolute URL
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("redirect_to must be an absolute URL")
	}

	// Only allow HTTPS (except for localhost)
	if u.Scheme != "https" {
		if !util.IsLocalhost(u.Host) {
			return fmt.Errorf("redirect_to must use HTTPS scheme (except for localhost)")
		}
	}

	// Must not contain credentials
	if u.User != nil {
		return fmt.Errorf("redirect_to must not contain credentials")
	}

	// Validate against trusted origins
	if len(trustedOrigins) == 0 {
		return fmt.Errorf("no trusted origins configured")
	}

	origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	if !util.IsTrustedOrigin(origin, trustedOrigins) {
		return fmt.Errorf("redirect_to does not match any trusted origin")
	}

	return nil
}
