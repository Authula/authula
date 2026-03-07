package services

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	"strings"

	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

const backupCodeLength = 12

type BackupCodeService struct {
	Count           int
	PasswordService rootservices.PasswordService
}

func NewBackupCodeService(count int, passwordService rootservices.PasswordService) *BackupCodeService {
	return &BackupCodeService{Count: count, PasswordService: passwordService}
}

// Generate creates a set of cryptographically random backup codes using base32 encoding.
func (s *BackupCodeService) Generate() ([]string, error) {
	codes := make([]string, s.Count)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	// base32 encodes 5 bits per char; we need ceil(12*5/8) = 8 bytes to get at least 12 chars
	byteLen := (backupCodeLength*5 + 7) / 8
	for i := range codes {
		b := make([]byte, byteLen)
		if _, err := io.ReadFull(rand.Reader, b); err != nil {
			return nil, err
		}
		codes[i] = strings.ToLower(enc.EncodeToString(b)[:backupCodeLength])
	}
	return codes, nil
}

// HashCodes hashes each plaintext backup code using Argon2 via PasswordService.
func (s *BackupCodeService) HashCodes(codes []string) ([]string, error) {
	hashed := make([]string, len(codes))
	for i, code := range codes {
		h, err := s.PasswordService.Hash(code)
		if err != nil {
			return nil, err
		}
		hashed[i] = h
	}
	return hashed, nil
}

// VerifyAndConsume checks if code matches any hashed code and returns the remaining hashes.
func (s *BackupCodeService) VerifyAndConsume(hashedCodes []string, code string) ([]string, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	for i, h := range hashedCodes {
		if s.PasswordService.Verify(code, h) {
			remaining := make([]string, 0, len(hashedCodes)-1)
			remaining = append(remaining, hashedCodes[:i]...)
			remaining = append(remaining, hashedCodes[i+1:]...)
			return remaining, true
		}
	}
	return hashedCodes, false
}
