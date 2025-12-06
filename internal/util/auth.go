package util

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

func ExtractProviderName(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := range parts {
		if parts[i] == "oauth2" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func GetCookieOptions(cfg *domain.Config) (isSecure bool, sameSite http.SameSite) {
	isSecure = strings.HasPrefix(cfg.BaseURL, "https")
	sameSite = http.SameSiteLaxMode
	if isSecure {
		sameSite = http.SameSiteNoneMode
	}
	return isSecure, sameSite
}

func AppendQueryParam(originalURL string, key string, value string) string {
	URL, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}
	q := URL.Query()
	q.Set(key, value)
	URL.RawQuery = q.Encode()
	return URL.String()
}

func IsTrustedRedirect(target string, trusted []string) bool {
	// Allow only safe relative paths
	if strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//") {
		return true
	}

	// Parse absolute URL
	u, err := url.Parse(target)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	// Compare with trusted origins
	for _, origin := range trusted {
		if strings.EqualFold(origin, u.Scheme+"://"+u.Host) {
			return true
		}
	}

	return false
}
