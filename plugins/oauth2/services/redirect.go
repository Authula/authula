package services

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Authula/authula/util"
)

// ValidateRedirectURL validates an OAuth2 provider redirect URL registered with
// the provider. Redirect URIs must be absolute, exact, and use HTTPS outside
// localhost. Wildcards, fragments, credentials, and relative paths are rejected.
func ValidateRedirectURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("redirect_url is required")
	}
	if strings.Contains(raw, "*") {
		return fmt.Errorf("redirect_url must not contain wildcards")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("redirect_url must be a valid absolute URL")
	}
	if parsed.Opaque != "" || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("redirect_url must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("redirect_url must use http or https")
	}
	if parsed.User != nil {
		return fmt.Errorf("redirect_url must not contain credentials")
	}
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Errorf("redirect_url must not contain a fragment")
	}
	if parsed.Scheme == "http" && !util.IsLocalhost(parsed.Host) {
		return fmt.Errorf("redirect_url must use https except for localhost")
	}

	return nil
}

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
