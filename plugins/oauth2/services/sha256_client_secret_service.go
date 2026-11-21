package services

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"github.com/Authula/authula/util"
)

const (
	clientSecretRandomLength = 48
	clientSecretSaltLength   = 16
	clientSecretPrefixLength = 12
)

type sha256ClientSecretService struct {
	prefix string
}

// NewSHA256ClientSecretService returns a client secret service that prefixes
// generated secrets with prefix and stores salted SHA-256 hashes.
func NewSHA256ClientSecretService(prefix string) ClientSecretService {
	return &sha256ClientSecretService{prefix: prefix}
}

func (s *sha256ClientSecretService) Generate() (string, error) {
	random, err := util.GenerateRandomString(clientSecretRandomLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate client secret: %w", err)
	}
	return s.prefix + random, nil
}

func (s *sha256ClientSecretService) Hash(secret string) (string, string, error) {
	saltBytes := make([]byte, clientSecretSaltLength)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate client secret salt: %w", err)
	}

	sum := sha256.Sum256(append(saltBytes, []byte(secret)...))
	return base64.RawStdEncoding.EncodeToString(sum[:]), base64.RawStdEncoding.EncodeToString(saltBytes), nil
}

func (s *sha256ClientSecretService) Verify(secret, hash, salt string) bool {
	if secret == "" || hash == "" || salt == "" {
		return false
	}

	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}

	sum := sha256.Sum256(append(saltBytes, []byte(secret)...))
	return subtle.ConstantTimeCompare(expected, sum[:]) == 1
}

func (s *sha256ClientSecretService) Prefix(secret string) string {
	if len(secret) <= clientSecretPrefixLength {
		return secret
	}
	return secret[:clientSecretPrefixLength]
}
