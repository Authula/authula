package services

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

type Argon2PasswordService struct{}

func NewArgon2PasswordService() *Argon2PasswordService {
	return &Argon2PasswordService{}
}

func (p *Argon2PasswordService) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	full := append(salt, hash...)
	return base64.RawStdEncoding.EncodeToString(full), nil
}

func (p *Argon2PasswordService) Verify(password, encoded string) bool {
	data, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(data) < 16 {
		return false
	}
	salt := data[:16]
	hash := data[16:]
	computed := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	if len(computed) != len(hash) {
		return false
	}
	for i := range hash {
		if computed[i] != hash[i] {
			return false
		}
	}
	return true
}
