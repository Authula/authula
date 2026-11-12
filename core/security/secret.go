package security

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// SecretKeySize is the required decoded length of an Authula secret. It matches
// the XChaCha20-Poly1305 key size used for token encryption.
const SecretKeySize = 32

// ErrInvalidSecret indicates that a secret is empty, contains whitespace, is
// not standard padded Base64, or does not decode to exactly SecretKeySize bytes.
var ErrInvalidSecret = errors.New("invalid secret")

// DecodeSecret decodes a standard padded Base64 secret.
func DecodeSecret(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: value must not be empty", ErrInvalidSecret)
	}

	// Go's Base64 decoder ignores \r and \n inside input, so reject all
	// remaining whitespace explicitly.
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			return nil, fmt.Errorf("%w: value must not contain whitespace", ErrInvalidSecret)
		}
	}

	raw, err := base64.StdEncoding.Strict().DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: value must be standard padded Base64: %v", ErrInvalidSecret, err)
	}
	if len(raw) != SecretKeySize {
		return nil, fmt.Errorf("%w: value must decode to %d bytes, got %d", ErrInvalidSecret, SecretKeySize, len(raw))
	}

	return raw, nil
}
