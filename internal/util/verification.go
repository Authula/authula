package util

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildVerificationURL constructs the verification endpoint URL with optional callback.
func BuildVerificationURL(baseURL string, basePath string, token string, callbackURL *string) string {
	return BuildActionURL(baseURL, basePath, "/verify-email", token, callbackURL)
}

// BuildActionURL centralizes URL building logic for token-based flows while preserving callback semantics.
func BuildActionURL(baseURL string, basePath string, actionPath string, token string, callbackURL *string) string {
	urlToConstruct := buildAbsoluteActionURL(baseURL, basePath, actionPath)
	urlObj, _ := url.Parse(urlToConstruct)
	q := urlObj.Query()
	q.Set("token", token)

	if callbackURL != nil && *callbackURL != "" {
		q.Set("callback_url", *callbackURL)
	}

	urlObj.RawQuery = q.Encode()
	return urlObj.String()
}

func buildAbsoluteActionURL(baseURL, basePath, actionPath string) string {
	base := strings.TrimRight(baseURL, "/")
	pathSegment := strings.Trim(basePath, "/")
	actionSegment := strings.TrimLeft(actionPath, "/")

	switch {
	case pathSegment == "" && actionSegment == "":
		return base
	case pathSegment == "":
		return fmt.Sprintf("%s/%s", base, actionSegment)
	default:
		return fmt.Sprintf("%s/%s/%s", base, pathSegment, actionSegment)
	}
}
