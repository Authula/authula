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

func generatedBackupCodes(t *testing.T, count int) ([]string, *BackupCodeService) {
	t.Helper()

	svc := newTestBackupCodeService(count)
	codes, err := svc.Generate()
	require.NoError(t, err)
	return codes, svc
}

func TestNewBackupCodeService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
	}{
		{
			name:  "initializes count and password service",
			count: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			passwordService := coreservices.NewArgon2PasswordService()
			svc := NewBackupCodeService(tt.count, passwordService)

			require.NotNil(t, svc)
			assert.Equal(t, tt.count, svc.Count)
			assert.Same(t, passwordService, svc.PasswordService)
		})
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
		check func(t *testing.T, codes []string, svc *BackupCodeService)
	}{
		{
			name:  "generate valid lowercase base32 codes",
			count: 10,
			check: func(t *testing.T, codes []string, svc *BackupCodeService) {
				require.Len(t, codes, 10)

				seen := make(map[string]struct{}, len(codes))
				for _, code := range codes {
					assert.Len(t, code, 12)
					assert.Equal(t, strings.ToLower(code), code, "code should be lowercase")
					for _, ch := range code {
						valid := (ch >= 'a' && ch <= 'z') || (ch >= '2' && ch <= '7')
						assert.True(t, valid, "invalid base32 char: %c", ch)
					}
					if _, exists := seen[code]; exists {
						assert.Failf(t, "duplicate backup code", "duplicate backup code: %s", code)
					}
					seen[code] = struct{}{}
				}
			},
		},
		{
			name:  "generate zero codes",
			count: 0,
			check: func(t *testing.T, codes []string, svc *BackupCodeService) {
				require.Empty(t, codes)

				hashed, err := svc.HashCodes(codes)
				require.NoError(t, err)
				require.Empty(t, hashed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codes, svc := generatedBackupCodes(t, tt.count)
			tt.check(t, codes, svc)
		})
	}
}

func TestHashCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
		check func(t *testing.T, codes, hashed []string)
	}{
		{
			name:  "hashes each code to a different value",
			count: 10,
			check: func(t *testing.T, codes, hashed []string) {
				require.Len(t, hashed, 10)

				for i := range codes {
					assert.NotEqual(t, codes[i], hashed[i])
				}
			},
		},
		{
			name:  "handles empty input",
			count: 0,
			check: func(t *testing.T, codes, hashed []string) {
				require.Empty(t, hashed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestBackupCodeService(tt.count)

			var codes []string
			if tt.count > 0 {
				var err error
				codes, err = svc.Generate()
				require.NoError(t, err)
			}

			hashed, err := svc.HashCodes(codes)
			require.NoError(t, err)
			tt.check(t, codes, hashed)
		})
	}
}

func TestVerifyAndConsumeBackupCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
		input func(codes []string) string
		check func(t *testing.T, remaining []string, ok bool)
	}{
		{
			name:  "matches exact code",
			count: 10,
			input: func(codes []string) string {
				return codes[3]
			},
			check: func(t *testing.T, remaining []string, ok bool) {
				assert.True(t, ok)
				assert.Len(t, remaining, 9)
			},
		},
		{
			name:  "rejects invalid code",
			count: 10,
			input: func(codes []string) string {
				return "invalid-code"
			},
			check: func(t *testing.T, remaining []string, ok bool) {
				assert.False(t, ok)
				assert.Len(t, remaining, 10)
			},
		},
		{
			name:  "matches case-insensitively",
			count: 10,
			input: func(codes []string) string {
				return strings.ToUpper(codes[3])
			},
			check: func(t *testing.T, remaining []string, ok bool) {
				assert.True(t, ok, "expected case-insensitive match")
				assert.Len(t, remaining, 9)
			},
		},
		{
			name:  "trims whitespace",
			count: 10,
			input: func(codes []string) string {
				return "  " + codes[0] + "  "
			},
			check: func(t *testing.T, remaining []string, ok bool) {
				assert.True(t, ok, "expected whitespace-trimmed match")
				assert.Len(t, remaining, 9)
			},
		},
		{
			name:  "returns original hashes when input is empty",
			count: 0,
			input: func(codes []string) string {
				return "anything"
			},
			check: func(t *testing.T, remaining []string, ok bool) {
				assert.False(t, ok)
				assert.Empty(t, remaining)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestBackupCodeService(tt.count)

			var codes []string
			if tt.count > 0 {
				var err error
				codes, err = svc.Generate()
				require.NoError(t, err)
			}

			var hashed []string
			if len(codes) > 0 {
				var err error
				hashed, err = svc.HashCodes(codes)
				require.NoError(t, err)
			}

			remaining, ok := svc.VerifyAndConsume(hashed, tt.input(codes))
			tt.check(t, remaining, ok)
		})
	}
}
