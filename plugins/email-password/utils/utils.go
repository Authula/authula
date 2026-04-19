package utils

import (
	internalutils "github.com/Authula/authula/internal/util"
)

func BuildVerificationURL(baseURL string, basePath string, token string, callbackURL *string) string {
	return internalutils.BuildActionURL(baseURL, basePath, "/email-password/verify-email", token, callbackURL)
}
