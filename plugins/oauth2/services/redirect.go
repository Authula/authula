package services

import (
	"fmt"
	"net"
	"net/url"
	"strings"
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
		if !isLocalhost(u.Host) {
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
	if !matchesTrustedOrigin(origin, trustedOrigins) {
		return fmt.Errorf("redirect_to does not match any trusted origin")
	}

	return nil
}

// matchesTrustedOrigin checks if an origin matches any trusted origin pattern
func matchesTrustedOrigin(origin string, trustedOrigins []string) bool {
	for _, trusted := range trustedOrigins {
		if matchesOriginPattern(origin, trusted) {
			return true
		}
	}
	return false
}

// matchesOriginPattern checks if an origin matches a pattern (supports wildcards)
func matchesOriginPattern(origin string, pattern string) bool {
	// Exact match
	if origin == pattern {
		return true
	}

	// Wildcard subdomain match (e.g., https://*.example.com)
	if strings.HasPrefix(pattern, "*.") {
		patternDomain := strings.TrimPrefix(pattern, "*.")
		originURL, err := url.Parse(origin)
		if err != nil {
			return false
		}

		// Check if the origin host ends with the pattern domain
		if strings.HasSuffix(originURL.Host, patternDomain) {
			return true
		}
	}

	return false
}

// isLocalhost checks if a host is localhost
func isLocalhost(host string) bool {
	// Remove port if present
	hostname := strings.Split(host, ":")[0]

	// Check for common localhost patterns
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return true
	}

	// Check if it's an IP address that resolves to localhost
	ip := net.ParseIP(hostname)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}
