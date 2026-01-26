package repositories

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// CryptoTokenRepositoryImpl implements TokenRepository using modern crypto
type CryptoTokenRepositoryImpl struct {
	secretKey []byte
}

// NewCryptoTokenRepository creates a new crypto token repository
// secret must be at least 32 bytes of high-entropy data
func NewCryptoTokenRepository(secret string) TokenRepository {
	key := sha256.Sum256([]byte(secret)) // normalize to 32 bytes

	return &CryptoTokenRepositoryImpl{
		secretKey: key[:],
	}
}

// Generate generates a random 256-bit token encoded as hex
func (r *CryptoTokenRepositoryImpl) Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Hash creates a SHA256 hash of the token (for DB storage)
func (r *CryptoTokenRepositoryImpl) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Encrypt encrypts a token
// Format: base64url([nonce_24][ciphertext+tag])
func (r *CryptoTokenRepositoryImpl) Encrypt(token string) (string, error) {
	aead, err := chacha20poly1305.NewX(r.secretKey)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nil, nonce, []byte(token), nil)

	out := append(nonce, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(out), nil
}

// Decrypt decrypts a token
func (r *CryptoTokenRepositoryImpl) Decrypt(encrypted string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	if len(raw) < chacha20poly1305.NonceSizeX {
		return "", errors.New("invalid encrypted token")
	}

	nonce := raw[:chacha20poly1305.NonceSizeX]
	ciphertext := raw[chacha20poly1305.NonceSizeX:]

	aead, err := chacha20poly1305.NewX(r.secretKey)
	if err != nil {
		return "", err
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("invalid or tampered token")
	}

	return string(plaintext), nil
}
