package repositories

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"unicode"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/Authula/authula/models"
)

// cryptoTokenRepository implements TokenRepository using XChaCha20-Poly1305
// with an ordered keyring for non-destructive key rotation.
type cryptoTokenRepository struct {
	current    cipher.AEAD
	decryptors []cipher.AEAD
}

// NewCryptoTokenRepository creates a token repository from Base64-encoded
// 32-byte keys. The first configured key encrypts new values; every configured
// key is tried when decrypting. When secrets is empty, secret is both the
// current and the only decryption key.
func NewCryptoTokenRepository(secret string, secrets []models.SecretConfig) (TokenRepository, error) {
	if len(secrets) == 0 {
		aead, err := parseSecretKey(secret)
		if err != nil {
			return nil, err
		}

		return &cryptoTokenRepository{
			current:    aead,
			decryptors: []cipher.AEAD{aead},
		}, nil
	}

	decryptors := make([]cipher.AEAD, 0, len(secrets)+1)
	var current cipher.AEAD

	for i, entry := range secrets {
		aead, err := parseSecretKey(entry.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: secrets entry %d (version %d): %v", ErrInvalidSecret, i+1, entry.Version, err)
		}
		if i == 0 {
			current = aead
		}
		decryptors = append(decryptors, aead)
	}

	// The singular secret remains a final decryption fallback so that values
	// written before a keyring was configured stay readable.
	if secret != "" {
		aead, err := parseSecretKey(secret)
		if err != nil {
			return nil, fmt.Errorf("%w: singular fallback secret: %v", ErrInvalidSecret, err)
		}
		decryptors = append(decryptors, aead)
	}

	return &cryptoTokenRepository{
		current:    current,
		decryptors: decryptors,
	}, nil
}

// parseSecretKey decodes and validates a Base64-encoded 32-byte key.
func parseSecretKey(secret string) (cipher.AEAD, error) {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: key must not be empty", ErrInvalidSecret)
	}

	// Go's Base64 decoder ignores \r and \n inside input, so reject all
	// remaining whitespace explicitly.
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			return nil, fmt.Errorf("%w: key must not contain whitespace", ErrInvalidSecret)
		}
	}

	raw, err := base64.StdEncoding.Strict().DecodeString(trimmed)
	if err != nil {
		raw, err = base64.RawStdEncoding.Strict().DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%w: key must be standard Base64: %v", ErrInvalidSecret, err)
		}
	}

	if len(raw) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("%w: key must decode to %d bytes, got %d", ErrInvalidSecret, chacha20poly1305.KeySize, len(raw))
	}

	aead, err := chacha20poly1305.NewX(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSecret, err)
	}

	return aead, nil
}

// Generate generates a random 256-bit token encoded as hex.
func (r *cryptoTokenRepository) Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Hash creates a SHA256 hash of the token (for DB storage).
func (r *cryptoTokenRepository) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Encrypt encrypts a token with the current key.
// Format: base64url(nonce || ciphertext || tag).
func (r *cryptoTokenRepository) Encrypt(token string) (string, error) {
	nonce := make([]byte, r.current.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	out := make([]byte, 0, len(nonce)+len(token)+r.current.Overhead())
	out = append(out, nonce...)
	out = r.current.Seal(out, nonce, []byte(token), nil)

	return base64.RawURLEncoding.EncodeToString(out), nil
}

// Decrypt decrypts a token by trying every configured key in order. It never
// re-encrypts or writes to storage.
func (r *cryptoTokenRepository) Decrypt(encrypted string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidEncryptedToken, err)
	}

	nonceSize := r.current.NonceSize()
	if len(raw) < nonceSize+r.current.Overhead() {
		return "", ErrInvalidEncryptedToken
	}

	nonce := raw[:nonceSize]
	ciphertext := raw[nonceSize:]

	for _, aead := range r.decryptors {
		plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
		if err == nil {
			return string(plaintext), nil
		}
	}

	return "", ErrTokenTampered
}
