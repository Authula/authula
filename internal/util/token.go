package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// DefaultTokenBytes is the recommended default length (in bytes) for generated tokens.
// 32 bytes = 256 bits of entropy which is suitable for most token use-cases.
const DefaultTokenBytes = 32

// GenerateRandomBytes returns n cryptographically secure random bytes.
// Returns an error if n <= 0 or if the random source fails.
func GenerateRandomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("invalid length %d", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return b, nil
}

// GenerateRandomTokenBase64URL returns a URL-safe base64 (raw, no padding) encoded token
// produced from n random bytes. Use DefaultTokenBytes for a sensible default.
func GenerateRandomTokenBase64URL(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateRandomTokenHex returns a hex-encoded token produced from n random bytes.
// The result length will be 2*n characters.
func GenerateRandomTokenHex(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateToken returns a default-sized (DefaultTokenBytes) URL-safe base64 token.
func GenerateToken() (string, error) {
	return GenerateRandomTokenBase64URL(DefaultTokenBytes)
}

// EncryptToken encrypts the token using AES-256-GCM with the provided secret.
// Returns the base64-encoded encrypted token.
func EncryptToken(token string, secret string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token is required")
	}
	if secret == "" {
		return "", fmt.Errorf("secret is required")
	}

	// Derive 32-byte key from secret using SHA256
	keyHash := sha256.Sum256([]byte(secret))
	key := keyHash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt token
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)

	// Return base64-encoded ciphertext
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken decrypts an AES-256-GCM encrypted token.
// Expects token in base64-encoded format.
func DecryptToken(encryptedToken string, secret string) (string, error) {
	if encryptedToken == "" {
		return "", fmt.Errorf("encrypted token is required")
	}
	if secret == "" {
		return "", fmt.Errorf("secret is required")
	}

	// Decode base64
	ciphertext, err := base64.RawURLEncoding.DecodeString(encryptedToken)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	// Derive key from secret
	keyHash := sha256.Sum256([]byte(secret))
	key := keyHash[:]

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	// Decrypt token
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// HashTokenWithSecret creates an HMAC-SHA256 hash of the token using the secret.
// This is more secure than simple SHA256 hashing for token storage.
func HashTokenWithSecret(token string, secret string) string {
	return HMACSHA256([]byte(secret), []byte(token))
}
