package services

import (
	"github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

// TokenServiceImpl implements TokenService
type TokenServiceImpl struct {
	tokenRepo repositories.TokenRepository
}

// NewTokenService creates a new instance of TokenServiceImpl
func NewTokenService(tokenRepo repositories.TokenRepository) services.TokenService {
	return &TokenServiceImpl{
		tokenRepo: tokenRepo,
	}
}

// Generate generates a new token by delegating to the repository
func (t *TokenServiceImpl) Generate() (string, error) {
	return t.tokenRepo.Generate()
}

// Hash hashes the token by delegating to the repository
func (t *TokenServiceImpl) Hash(token string) string {
	return t.tokenRepo.Hash(token)
}

// Encrypt encrypts the token by delegating to the repository
func (t *TokenServiceImpl) Encrypt(token string) (string, error) {
	return t.tokenRepo.Encrypt(token)
}

// Decrypt decrypts the token by delegating to the repository
func (t *TokenServiceImpl) Decrypt(encryptedToken string) (string, error) {
	return t.tokenRepo.Decrypt(encryptedToken)
}
