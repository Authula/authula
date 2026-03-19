package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"time"
)

const (
	// The divisor for 6 digits (10^6)
	totpDivisor = 1_000_000
)

type TOTPService struct {
	Digits        int
	PeriodSeconds int
}

func NewTOTPService(digits, periodSeconds int) *TOTPService {
	return &TOTPService{Digits: digits, PeriodSeconds: periodSeconds}
}

// GenerateSecret generates a 20-byte cryptographically random secret, base32 encoded.
func (s *TOTPService) GenerateSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateCode generates a TOTP code for the given secret and time (RFC 6238).
func (s *TOTPService) GenerateCode(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", err
	}
	counter := uint64(t.Unix()) / uint64(s.PeriodSeconds)
	return s.hotp(key, counter), nil
}

// ValidateCode validates a TOTP code against the current time +/-1 window.
func (s *TOTPService) ValidateCode(secret, code string, t time.Time) bool {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false
	}
	counter := uint64(t.Unix()) / uint64(s.PeriodSeconds)
	for i := int64(-1); i <= 1; i++ {
		c := uint64(int64(counter) + i)
		if s.hotp(key, c) == code {
			return true
		}
	}
	return false
}

// BuildURI builds an otpauth:// URI for QR code display.
func (s *TOTPService) BuildURI(secret, issuer, email string) string {
	label := url.PathEscape(issuer) + ":" + url.PathEscape(email)
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", issuer)
	v.Set("digits", fmt.Sprintf("%d", s.Digits))
	v.Set("period", fmt.Sprintf("%d", s.PeriodSeconds))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, v.Encode())
}

// hotp computes HOTP per RFC 4226.
func (s *TOTPService) hotp(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	// Use SHA-1 for universal compatibility
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	// Dynamic Truncation
	offset := sum[len(sum)-1] & 0xf

	code := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)

	code = code % totpDivisor

	return fmt.Sprintf("%06d", code)
}
