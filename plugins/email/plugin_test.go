package email

import (
	"strings"
	"testing"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email/constants"
	emailtypes "github.com/Authula/authula/plugins/email/types"
)

func newTestPluginContext() *models.PluginContext {
	return &models.PluginContext{
		Logger:          &internaltests.MockLogger{},
		ServiceRegistry: &internaltests.TestServiceRegistry{},
		GetConfig:       func() *models.Config { return &models.Config{} },
	}
}

func setProviderCredentials(t *testing.T) {
	t.Helper()
	t.Setenv(constants.EnvSMTPHost, "smtp.example.com")
	t.Setenv(constants.EnvSMTPPort, "587")
	t.Setenv(constants.EnvResendApiKey, "re_test_key")
}

func TestEmailPlugin_Init_ResolvesProvider(t *testing.T) {
	tests := []struct {
		name           string
		envProvider    string
		configProvider emailtypes.EmailProviderType
		want           emailtypes.EmailProviderType
		wantErr        bool
		errContains    string
	}{
		{
			name:           "uses the config provider when the env var is unset",
			configProvider: emailtypes.ProviderSMTP,
			want:           emailtypes.ProviderSMTP,
		},
		{
			name:           "env var overrides the config provider",
			envProvider:    "resend",
			configProvider: emailtypes.ProviderSMTP,
			want:           emailtypes.ProviderResend,
		},
		{
			name:        "env var is used when no config provider is set",
			envProvider: "resend",
			want:        emailtypes.ProviderResend,
		},
		{
			name:           "env var value is normalised",
			envProvider:    "  SMTP ",
			configProvider: emailtypes.ProviderResend,
			want:           emailtypes.ProviderSMTP,
		},
		{
			name:           "blank env var falls back to the config provider",
			envProvider:    "   ",
			configProvider: emailtypes.ProviderResend,
			want:           emailtypes.ProviderResend,
		},
		{
			name:           "rejects an unsupported env var value",
			envProvider:    "sendgrid",
			configProvider: emailtypes.ProviderSMTP,
			wantErr:        true,
			errContains:    constants.EnvEmailProvider,
		},
		{
			name:        "errors when no provider is configured at all",
			wantErr:     true,
			errContains: constants.EnvEmailProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setProviderCredentials(t)
			t.Setenv(constants.EnvEmailProvider, tt.envProvider)

			plugin := New(emailtypes.EmailPluginConfig{
				Enabled:     true,
				Provider:    tt.configProvider,
				FromAddress: "noreply@example.com",
			})

			err := plugin.Init(newTestPluginContext())

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Init() expected an error, got nil (provider = %q)", plugin.PluginConfig.Provider)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Init() error = %q, want it to mention %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Init() returned unexpected error: %v", err)
			}
			if plugin.PluginConfig.Provider != tt.want {
				t.Errorf("Provider = %q, want %q", plugin.PluginConfig.Provider, tt.want)
			}
		})
	}
}

func TestEmailPlugin_Init_NormalisesFallbackProvider(t *testing.T) {
	setProviderCredentials(t)
	t.Setenv(constants.EnvEmailProvider, "smtp")

	plugin := New(emailtypes.EmailPluginConfig{
		Enabled:          true,
		Provider:         emailtypes.ProviderResend,
		FallbackProvider: "SMTP",
		FromAddress:      "noreply@example.com",
	})

	if err := plugin.Init(newTestPluginContext()); err != nil {
		t.Fatalf("Init() returned unexpected error: %v", err)
	}

	if plugin.PluginConfig.FallbackProvider != emailtypes.ProviderSMTP {
		t.Errorf("FallbackProvider = %q, want %q", plugin.PluginConfig.FallbackProvider, emailtypes.ProviderSMTP)
	}

	// The env var promoted SMTP to primary, so it must not also be the fallback.
	if plugin.EmailService.fallbackProvider != nil {
		t.Error("fallback provider should not be initialised when it matches the primary provider")
	}
}
