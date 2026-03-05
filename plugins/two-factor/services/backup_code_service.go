package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"strings"
)

type BackupCodeService struct {
	Count  int
	Length int
}

func NewBackupCodeService(count, length int) *BackupCodeService {
	return &BackupCodeService{Count: count, Length: length}
}

// Generate creates a set of cryptographically random backup codes.
func (s *BackupCodeService) Generate() ([]string, error) {
	codes := make([]string, s.Count)
	byteLen := (s.Length + 1) / 2 // hex encoding doubles length
	for i := range codes {
		b := make([]byte, byteLen)
		if _, err := io.ReadFull(rand.Reader, b); err != nil {
			return nil, err
		}
		codes[i] = hex.EncodeToString(b)[:s.Length]
	}
	return codes, nil
}

// VerifyAndConsume checks if code is in the list and returns the list without it.
// The comparison is constant-time and case-insensitive.
func (s *BackupCodeService) VerifyAndConsume(codes []string, code string) ([]string, bool) {
	code = strings.ToLower(strings.TrimSpace(code))
	for i, c := range codes {
		if subtle.ConstantTimeCompare([]byte(c), []byte(code)) == 1 {
			remaining := make([]string, 0, len(codes)-1)
			remaining = append(remaining, codes[:i]...)
			remaining = append(remaining, codes[i+1:]...)
			return remaining, true
		}
	}
	return codes, false
}
