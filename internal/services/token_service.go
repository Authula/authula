package services

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// TokenServiceImpl manages token operations using the application secret.
// This service uses Config.Secret for signing, encryption, and hashing tokens.
type TokenServiceImpl struct {
	config *models.Config
}

// NewTokenServiceImpl creates a new TokenServiceImpl with the provided config.
func NewTokenServiceImpl(config *models.Config) *TokenServiceImpl {
	return &TokenServiceImpl{
		config: config,
	}
}

// GenerateToken generates a new cryptographically secure random token.
func (ts *TokenServiceImpl) GenerateToken() (string, error) {
	return util.GenerateToken()
}

// HashToken creates a hash of the token using the application secret.
// This is more secure than plain SHA256 hashing for token storage.
func (ts *TokenServiceImpl) HashToken(token string) string {
	return util.HashTokenWithSecret(token, ts.config.Secret)
}

// GenerateEncryptedToken generates a token and encrypts it with the application secret.
func (ts *TokenServiceImpl) GenerateEncryptedToken() (string, error) {
	token, err := ts.GenerateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	if ts.config.Secret == "" {
		return "", fmt.Errorf("secret is required for token encryption")
	}

	encryptedToken, err := util.EncryptToken(token, ts.config.Secret)
	if err != nil {
		return "", fmt.Errorf("encrypt token: %w", err)
	}

	return encryptedToken, nil
}

// EncryptToken encrypts a plain token using the application secret.
func (ts *TokenServiceImpl) EncryptToken(token string) (string, error) {
	if ts.config.Secret == "" {
		return "", fmt.Errorf("secret is required for token encryption")
	}

	encryptedToken, err := util.EncryptToken(token, ts.config.Secret)
	if err != nil {
		return "", fmt.Errorf("encrypt token: %w", err)
	}

	return encryptedToken, nil
}

// DecryptToken decrypts an encrypted token using the application secret.
func (ts *TokenServiceImpl) DecryptToken(encryptedToken string) (string, error) {
	if ts.config.Secret == "" {
		return "", fmt.Errorf("secret is required for token decryption")
	}

	token, err := util.DecryptToken(encryptedToken, ts.config.Secret)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}

	return token, nil
}
