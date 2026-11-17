package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmailPasswordPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                            string
		config                          EmailPasswordPluginConfig
		wantMinPasswordLength           int
		wantMaxPasswordLength           int
		wantEmailVerificationExpiresIn  time.Duration
		wantPasswordResetExpiresIn      time.Duration
		wantRequestEmailChangeExpiresIn time.Duration
	}{
		{
			name:                            "zero_config_receives_defaults",
			wantMinPasswordLength:           8,
			wantMaxPasswordLength:           128,
			wantEmailVerificationExpiresIn:  24 * time.Hour,
			wantPasswordResetExpiresIn:      time.Hour,
			wantRequestEmailChangeExpiresIn: time.Hour,
		},
		{
			name: "explicit_values_are_preserved",
			config: EmailPasswordPluginConfig{
				MinPasswordLength:           12,
				MaxPasswordLength:           64,
				EmailVerificationExpiresIn:  48 * time.Hour,
				PasswordResetExpiresIn:      2 * time.Hour,
				RequestEmailChangeExpiresIn: 3 * time.Hour,
			},
			wantMinPasswordLength:           12,
			wantMaxPasswordLength:           64,
			wantEmailVerificationExpiresIn:  48 * time.Hour,
			wantPasswordResetExpiresIn:      2 * time.Hour,
			wantRequestEmailChangeExpiresIn: 3 * time.Hour,
		},
		{
			name: "only_missing_values_receive_defaults",
			config: EmailPasswordPluginConfig{
				MinPasswordLength:          10,
				EmailVerificationExpiresIn: 30 * time.Minute,
			},
			wantMinPasswordLength:           10,
			wantMaxPasswordLength:           128,
			wantEmailVerificationExpiresIn:  30 * time.Minute,
			wantPasswordResetExpiresIn:      time.Hour,
			wantRequestEmailChangeExpiresIn: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.wantMinPasswordLength, config.MinPasswordLength)
			require.Equal(t, tt.wantMaxPasswordLength, config.MaxPasswordLength)
			require.Equal(t, tt.wantEmailVerificationExpiresIn, config.EmailVerificationExpiresIn)
			require.Equal(t, tt.wantPasswordResetExpiresIn, config.PasswordResetExpiresIn)
			require.Equal(t, tt.wantRequestEmailChangeExpiresIn, config.RequestEmailChangeExpiresIn)
		})
	}
}
