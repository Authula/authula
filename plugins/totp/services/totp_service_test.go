package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		digits int
		period int
	}{
		{
			name:   "generates a base32 secret",
			digits: 6,
			period: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewTOTPService(tt.digits, tt.period)
			secret, err := svc.GenerateSecret()
			require.NoError(t, err)
			assert.NotEmpty(t, secret)
			assert.Len(t, secret, 32)
		})
	}
}

func TestGenerateAndVerifyCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		digits int
		period int
	}{
		{
			name:   "generates and validates a current code",
			digits: 6,
			period: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewTOTPService(tt.digits, tt.period)
			secret, err := svc.GenerateSecret()
			require.NoError(t, err)

			now := time.Now()
			code, err := svc.GenerateCode(secret, now)
			require.NoError(t, err)
			assert.Len(t, code, tt.digits)

			assert.True(t, svc.ValidateCode(secret, code, now))
		})
	}
}

func TestValidateCodeRejectsWrongCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		digits int
		period int
		code   string
	}{
		{
			name:   "rejects an incorrect code",
			digits: 6,
			period: 30,
			code:   "000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewTOTPService(tt.digits, tt.period)
			secret, err := svc.GenerateSecret()
			require.NoError(t, err)

			assert.False(t, svc.ValidateCode(secret, tt.code, time.Now()))
		})
	}
}

func TestValidateCodeAcceptsAdjacentWindows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		digits int
		period int
		window time.Duration
	}{
		{
			name:   "accepts the previous time window",
			digits: 6,
			period: 30,
			window: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewTOTPService(tt.digits, tt.period)
			secret, err := svc.GenerateSecret()
			require.NoError(t, err)

			now := time.Now()
			past := now.Add(-tt.window)
			code, err := svc.GenerateCode(secret, past)
			require.NoError(t, err)

			assert.True(t, svc.ValidateCode(secret, code, now))
		})
	}
}

func TestBuildURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		digits int
		period int
		secret string
		issuer string
		email  string
	}{
		{
			name:   "builds a valid otpauth uri",
			digits: 6,
			period: 30,
			secret: "JBSWY3DPEHPK3PXP",
			issuer: "MyApp",
			email:  "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewTOTPService(tt.digits, tt.period)
			uri := svc.BuildURI(tt.secret, tt.issuer, tt.email)
			assert.Contains(t, uri, "otpauth://totp/")
			assert.Contains(t, uri, "secret="+tt.secret)
			assert.Contains(t, uri, "issuer="+tt.issuer)
			assert.Contains(t, uri, "digits=6")
			assert.Contains(t, uri, "period=30")
		})
	}
}
