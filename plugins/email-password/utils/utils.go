package utils

import (
	util "github.com/Authula/authula/util"
)

func BuildVerificationURL(baseURL string, basePath string, token string, callbackURL *string) string {
	return util.BuildActionURL(baseURL, basePath, "/email-password/verify-email", token, callbackURL)
}
