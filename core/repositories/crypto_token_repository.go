package repositories

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// cryptoTokenRepository implements TokenRepository using XChaCha20-Poly1305
// with an ordered keyring for non-destructive key rotation.
type cryptoTokenRepository struct {
	current    cipher.AEAD
	decryptors []cipher.AEAD
}

// NewCryptoTokenRepository creates a new token repository
func NewCryptoTokenRepository(currentKey []byte, decryptors [][]byte) (TokenRepository, error) {
	current, err := newAEAD(currentKey)
	if err != nil {
		return nil, fmt.Errorf("%w: current key: %v", ErrInvalidSecret, err)
	}

	if len(decryptors) == 0 {
		decryptors = [][]byte{currentKey}
	}

	aeads := make([]cipher.AEAD, 0, len(decryptors))
	for i, key := range decryptors {
		aead, err := newAEAD(key)
		if err != nil {
			return nil, fmt.Errorf("%w: decryption key %d: %v", ErrInvalidSecret, i+1, err)
		}
		aeads = append(aeads, aead)
	}

	return &cryptoTokenRepository{
		current:    current,
		decryptors: aeads,
	}, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", chacha20poly1305.KeySize, len(key))
	}

	return chacha20poly1305.NewX(key)
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

// Decrypt decrypts a token by trying every configured key in order.
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
