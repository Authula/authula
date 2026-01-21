package security

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"
)

type HMACSigner struct {
	secret []byte
}

func NewHMACSigner(secret string) *HMACSigner {
	return &HMACSigner{
		secret: []byte(secret),
	}
}

// Generate creates a cryptographically signed token.
func (s *HMACSigner) Generate(ctx context.Context, subject string) (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	payload := make([]byte, 0, 128)
	payload = append(payload, nonce...)
	payload = append(payload, []byte(time.Now().UTC().Format(time.RFC3339Nano))...)
	payload = append(payload, []byte(subject)...)

	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	signature := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(signature), nil
}

// Verify verifies a token signature against a payload.
func (s *HMACSigner) Verify(ctx context.Context, token string, payload []byte) (bool, error) {
	expectedMAC := hmac.New(sha256.New, s.secret)
	expectedMAC.Write(payload)
	expected := expectedMAC.Sum(nil)

	given, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false, err
	}

	if !hmac.Equal(given, expected) {
		return false, errors.New("invalid token signature")
	}

	return true, nil
}
