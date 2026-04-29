package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/Authula/authula/models"
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
		parsedURL, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("invalid trusted origin %q: %w", origin, err)
		}

		if parsedURL.Scheme == "" {
			return fmt.Errorf("trusted origin %q must include scheme (https:// or http://)", origin)
		}

		if parsedURL.Host == "" {
			return fmt.Errorf("trusted origin %q must include host", origin)
		}

		// Warn against localhost usage in non-HTTP schemes (unusual but possible misconfiguration)
		if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") && strings.Contains(parsedURL.Host, "localhost") {
			return fmt.Errorf("trusted origin %q uses non-standard scheme %q with localhost", origin, parsedURL.Scheme)
		}
	}

	return nil
}

// IsTrustedOrigin reports whether origin matches one of the trusted origins.
func IsTrustedOrigin(origin string, trustedOrigins []string) bool {
	for _, trustedOrigin := range trustedOrigins {
		if matchesOriginPattern(origin, trustedOrigin) {
			return true
		}
	}

	return false
}

func matchesOriginPattern(origin string, pattern string) bool {
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	patternURL, err := url.Parse(pattern)
	if err != nil {
		return false
	}

	if originURL.Scheme == patternURL.Scheme && originURL.Host == patternURL.Host {
		return true
	}

	if after, ok := strings.CutPrefix(patternURL.Hostname(), "*."); ok {
		if originURL.Scheme != patternURL.Scheme {
			return false
		}

		if patternURL.Port() != "" && originURL.Port() != patternURL.Port() {
			return false
		}

		originHost := originURL.Hostname()
		if originHost == after || strings.HasSuffix(originHost, "."+after) {
			return true
		}
	}

	return false
}

// IsTrustedCallbackURL validates callback destinations for redirect flows.
// It accepts relative paths and trusted absolute URLs, while rejecting unsafe schemes.
func IsTrustedCallbackURL(callbackURL string, trustedOrigins []string) (*url.URL, error) {
	if callbackURL == "" {
		return nil, fmt.Errorf("callback_url is required")
	}
	if strings.HasPrefix(callbackURL, "//") {
		return nil, fmt.Errorf("callback_url must not be protocol-relative")
	}

	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return nil, fmt.Errorf("invalid callback_url: %w", err)
	}
	if parsed.Scheme != "" && parsed.Host == "" && parsed.Opaque != "" {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("callback_url uses an unsafe scheme")
		}
	}
	if parsed.Scheme != "" && parsed.Host == "" && parsed.Opaque == "" {
		return nil, fmt.Errorf("callback_url must be an absolute URL or relative path")
	}
	if parsed.Scheme == "" && parsed.Host == "" {
		if !strings.HasPrefix(callbackURL, "/") {
			return nil, fmt.Errorf("callback_url must be an absolute URL or relative path")
		}
		return parsed, nil
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("callback_url must be an absolute URL or relative path")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("callback_url must not contain credentials")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("callback_url uses an unsafe scheme")
	}
	if parsed.Scheme == "https" && IsLocalhost(parsed.Host) {
		return parsed, nil
	}
	if len(trustedOrigins) == 0 {
		return nil, fmt.Errorf("no trusted origins configured")
	}
	origin := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	if !IsTrustedOrigin(origin, trustedOrigins) {
		return nil, fmt.Errorf("callback_url is not a trusted origin")
	}
	return parsed, nil
}

func IsLocalhost(host string) bool {
	hostname := host
	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		hostname = splitHost
	} else if strings.HasPrefix(hostname, "[") && strings.HasSuffix(hostname, "]") {
		hostname = strings.TrimPrefix(strings.TrimSuffix(hostname, "]"), "[")
	}

	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return true
	}

	ip := net.ParseIP(hostname)
	return ip != nil && ip.IsLoopback()
}

func ValidateTrustedHeadersAndProxies(logger models.Logger, trustedHeaders []string, trustedProxies []string) {
	if len(trustedHeaders) > 0 && len(trustedProxies) == 0 {
		logger.Warn(
			"Security Warning: TrustedHeaders are defined, but TrustedProxies is empty. " +
				"The headers will be ignored to prevent IP spoofing. " +
				"Add your proxy IP to 'trusted_proxies' to enable header extraction.",
		)
	}
}
