package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateRandomString generates a cryptographically secure random URL-safe string
// of exactly length characters.
func GenerateRandomString(length int) (string, error) {
	if length < 0 {
		return "", fmt.Errorf("length must not be negative")
	}
	if length == 0 {
		return "", nil
	}

	// Raw URL encoding produces ceil(4*n/3) characters for n bytes.
	byteLen := (length*3 + 3) / 4
	randomBytes := make([]byte, byteLen)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes)[:length], nil
}
