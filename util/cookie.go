package util

import (
	"net/http"
	"strings"
)

// ParseSameSite maps a configured SameSite string ("lax", "strict" or "none", case-insensitive)
// to its http.SameSite value. Unknown or empty values fall back to Lax.
func ParseSameSite(value string) http.SameSite {
	switch strings.ToLower(value) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
