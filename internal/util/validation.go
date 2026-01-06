package util

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func InitValidator() {
	Validate = validator.New()
}

func ValidateStruct(s any) error {
	return Validate.Struct(s)
}

// ValidateTrustedOrigins validates that all trusted origins are well-formed URLs
func ValidateTrustedOrigins(trustedOrigins []string) error {
	for _, origin := range trustedOrigins {
		// Parse as URL to validate format (scheme://host[:port])
		u, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("invalid trusted origin %q: %w", origin, err)
		}

		// Ensure scheme is present
		if u.Scheme == "" {
			return fmt.Errorf("trusted origin %q must include scheme (https:// or http://)", origin)
		}

		// Ensure host is present
		if u.Host == "" {
			return fmt.Errorf("trusted origin %q must include host", origin)
		}

		// Warn against localhost usage in non-HTTP schemes (unusual but possible misconfiguration)
		if (u.Scheme != "http" && u.Scheme != "https") && strings.Contains(u.Host, "localhost") {
			return fmt.Errorf("trusted origin %q uses non-standard scheme %q with localhost", origin, u.Scheme)
		}
	}

	return nil
}
