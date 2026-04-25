package services

import (
	"github.com/Authula/authula/internal/repositories"
	"github.com/Authula/authula/services"
)

type tokenService struct {
	tokenRepo repositories.TokenRepository
}

func NewTokenService(tokenRepo repositories.TokenRepository) services.TokenService {
	return &tokenService{
		tokenRepo: tokenRepo,
	}
}

func (t *tokenService) Generate() (string, error) {
	return t.tokenRepo.Generate()
}

func (t *tokenService) Hash(token string) string {
	return t.tokenRepo.Hash(token)
}

func (t *tokenService) Encrypt(token string) (string, error) {
	return t.tokenRepo.Encrypt(token)
}

func (t *tokenService) Decrypt(encryptedToken string) (string, error) {
	return t.tokenRepo.Decrypt(encryptedToken)
}
