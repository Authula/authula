package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

func TestMagicLinkPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config MagicLinkPluginConfig
		want   time.Duration
	}{
		{
			name: "zero_expiry_receives_default",
			want: 15 * time.Minute,
		},
		{
			name:   "negative_expiry_receives_default",
			config: MagicLinkPluginConfig{ExpiresIn: -time.Minute},
			want:   15 * time.Minute,
		},
		{
			name:   "explicit_expiry_is_preserved",
			config: MagicLinkPluginConfig{ExpiresIn: 30 * time.Minute},
			want:   30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config
			config.ApplyDefaults()

			require.Equal(t, tt.want, config.ExpiresIn)
		})
	}
}

func TestMagicLinkPluginConfig_JSONOmitsEmailSender(t *testing.T) {
	t.Parallel()

	config := MagicLinkPluginConfig{
		Enabled:                        true,
		ExpiresIn:                      time.Minute,
		DisableSignUp:                  true,
		SendMagicLinkVerificationEmail: func(SendMagicLinkVerificationEmailParams, *models.RequestContext) error { return nil },
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))
	require.Contains(t, fields, "enabled")
	require.Contains(t, fields, "expires_in")
	require.Contains(t, fields, "disable_sign_up")
	require.NotContains(t, fields, "SendMagicLinkVerificationEmail")
}
