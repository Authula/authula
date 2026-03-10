package services

import (
	"strings"
	"testing"

	coreservices "github.com/GoBetterAuth/go-better-auth/v2/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBackupCodeService(count int) *BackupCodeService {
	return NewBackupCodeService(count, coreservices.NewArgon2PasswordService())
}

func TestGenerateBackupCodes(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)
	assert.Len(t, codes, 10)
	for _, code := range codes {
		assert.Len(t, code, 12)
	}
}

func TestBackupCodesAreUnique(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)
	seen := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "duplicate backup code: %s", code)
		seen[code] = true
	}
}

func TestBackupCodesAreLowercaseBase32(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)
	for _, code := range codes {
		assert.Equal(t, strings.ToLower(code), code, "code should be lowercase")
		for _, ch := range code {
			valid := (ch >= 'a' && ch <= 'z') || (ch >= '2' && ch <= '7')
			assert.True(t, valid, "invalid base32 char: %c", ch)
		}
	}
}

func TestHashAndVerifyBackupCodes(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	hashed, err := svc.HashCodes(codes)
	require.NoError(t, err)
	assert.Len(t, hashed, 10)

	// Each hashed code should differ from the plaintext
	for i := range codes {
		assert.NotEqual(t, codes[i], hashed[i])
	}
}

func TestVerifyAndConsumeBackupCode(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	hashed, err := svc.HashCodes(codes)
	require.NoError(t, err)

	target := codes[3]
	remaining, ok := svc.VerifyAndConsume(hashed, target)
	assert.True(t, ok)
	assert.Len(t, remaining, 9)
}

func TestVerifyAndConsumeRejectsInvalid(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	hashed, err := svc.HashCodes(codes)
	require.NoError(t, err)

	remaining, ok := svc.VerifyAndConsume(hashed, "invalid-code")
	assert.False(t, ok)
	assert.Len(t, remaining, 10)
}

func TestVerifyAndConsumeCaseInsensitive(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	hashed, err := svc.HashCodes(codes)
	require.NoError(t, err)

	target := codes[3]
	upperTarget := strings.ToUpper(target)

	remaining, ok := svc.VerifyAndConsume(hashed, upperTarget)
	assert.True(t, ok, "expected case-insensitive match")
	assert.Len(t, remaining, 9)
}

func TestVerifyAndConsumeTrimsWhitespace(t *testing.T) {
	svc := newTestBackupCodeService(10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	hashed, err := svc.HashCodes(codes)
	require.NoError(t, err)

	target := codes[0]
	remaining, ok := svc.VerifyAndConsume(hashed, "  "+target+"  ")
	assert.True(t, ok, "expected whitespace-trimmed match")
	assert.Len(t, remaining, 9)
}
