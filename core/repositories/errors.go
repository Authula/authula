package repositories

import (
	"errors"

	coresecurity "github.com/Authula/authula/core/security"
)

var (
	// ErrInvalidSecret indicates that an encryption key is not exactly 32
	// bytes. Config load validates and decodes secrets, so this only surfaces
	// when a caller bypasses config validation.
	ErrInvalidSecret = coresecurity.ErrInvalidSecret
	// ErrInvalidEncryptedToken indicates that an encrypted value is not valid
	// Base64url or is too short to contain a nonce and an authentication tag.
	ErrInvalidEncryptedToken = errors.New("invalid encrypted token")
	// ErrTokenTampered indicates that no configured key could authenticate the
	// encrypted value.
	ErrTokenTampered = errors.New("invalid or tampered token")
)
