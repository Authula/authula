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
	clientSecretHintLength   = 4
)

type sha256ClientSecretService struct {
	prefix       string
	randomString func(int) (string, error)
	readRandom   func([]byte) (int, error)
}

// NewSHA256ClientSecretService returns a client secret service that prefixes
// generated secrets with prefix and stores salted SHA-256 hashes.
func NewSHA256ClientSecretService(prefix string) ClientSecretService {
	return &sha256ClientSecretService{
		prefix:       prefix,
		randomString: util.GenerateRandomString,
		readRandom:   rand.Read,
	}
}

func (s *sha256ClientSecretService) Generate() (*GeneratedClientSecret, error) {
	random, err := s.randomString(clientSecretRandomLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	value := s.prefix + random
	hash, salt, err := s.Hash(value)
	if err != nil {
		return nil, err
	}

	return &GeneratedClientSecret{
		Value:  value,
		Hash:   hash,
		Salt:   salt,
		Prefix: s.prefix,
		Start:  random[:clientSecretHintLength],
		Last:   random[len(random)-clientSecretHintLength:],
	}, nil
}

func (s *sha256ClientSecretService) Hash(secret string) (string, string, error) {
	saltBytes := make([]byte, clientSecretSaltLength)
	if _, err := s.readRandom(saltBytes); err != nil {
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
