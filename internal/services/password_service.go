package services

import (
	"github.com/alexedwards/argon2id"
)

type Argon2PasswordService struct{}

func NewArgon2PasswordService() *Argon2PasswordService {
	return &Argon2PasswordService{}
}

// HashPassword hashes the given password using Argon2id.
func (s *Argon2PasswordService) HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// VerifyPassword verifies the given password against the provided hash using Argon2id.
func (s *Argon2PasswordService) VerifyPassword(password string, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}
