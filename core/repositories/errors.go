package repositories

import "errors"

var (
	// ErrInvalidSecret indicates that a token-encryption key is empty,
	// contains whitespace, is not standard Base64, or does not decode to
	// exactly 32 bytes.
	ErrInvalidSecret = errors.New("invalid token repository secret")
	// ErrInvalidEncryptedToken indicates that an encrypted value is not valid
	// Base64url or is too short to contain a nonce and an authentication tag.
	ErrInvalidEncryptedToken = errors.New("invalid encrypted token")
	// ErrTokenTampered indicates that no configured key could authenticate the
	// encrypted value.
	ErrTokenTampered = errors.New("invalid or tampered token")
)
