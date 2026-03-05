package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBackupCodes(t *testing.T) {
	svc := NewBackupCodeService(10, 10)
	codes, err := svc.Generate()
	require.NoError(t, err)
	assert.Len(t, codes, 10)
	for _, code := range codes {
		assert.Len(t, code, 10)
	}
}

func TestBackupCodesAreUnique(t *testing.T) {
	svc := NewBackupCodeService(10, 10)
	codes, err := svc.Generate()
	require.NoError(t, err)
	seen := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "duplicate backup code: %s", code)
		seen[code] = true
	}
}

func TestVerifyAndConsumeBackupCode(t *testing.T) {
	svc := NewBackupCodeService(10, 10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	target := codes[3]
	remaining, ok := svc.VerifyAndConsume(codes, target)
	assert.True(t, ok)
	assert.Len(t, remaining, 9)
	assert.NotContains(t, remaining, target)
}

func TestVerifyAndConsumeRejectsInvalid(t *testing.T) {
	svc := NewBackupCodeService(10, 10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	remaining, ok := svc.VerifyAndConsume(codes, "invalid-code")
	assert.False(t, ok)
	assert.Len(t, remaining, 10)
}

func TestVerifyAndConsumeCaseInsensitive(t *testing.T) {
	svc := NewBackupCodeService(10, 10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	target := codes[3]
	upperTarget := strings.ToUpper(target)

	remaining, ok := svc.VerifyAndConsume(codes, upperTarget)
	assert.True(t, ok, "expected case-insensitive match")
	assert.Len(t, remaining, 9)
	assert.NotContains(t, remaining, target)
}

func TestVerifyAndConsumeTrimsWhitespace(t *testing.T) {
	svc := NewBackupCodeService(10, 10)
	codes, err := svc.Generate()
	require.NoError(t, err)

	target := codes[0]
	remaining, ok := svc.VerifyAndConsume(codes, "  "+target+"  ")
	assert.True(t, ok, "expected whitespace-trimmed match")
	assert.Len(t, remaining, 9)
}
