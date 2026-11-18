package email

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalplugins "github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email/constants"
	"github.com/Authula/authula/plugins/email/providers"
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

// isolateEmailEnv clears every environment variable the email plugin reads so
// tests never depend on the developer's shell.
func isolateEmailEnv(t *testing.T) {
	t.Helper()
	t.Setenv(constants.EnvEmailFrom, "")
	t.Setenv(constants.EnvEmailProvider, "")
	t.Setenv(constants.EnvSMTPHost, "")
	t.Setenv(constants.EnvSMTPPort, "")
	t.Setenv(constants.EnvSMTPUser, "")
	t.Setenv(constants.EnvSMTPPass, "")
	t.Setenv(constants.EnvResendApiKey, "")
}

func TestEmailPlugin_Metadata(t *testing.T) {
	meta := New(emailtypes.EmailPluginConfig{}).Metadata()

	assert.Equal(t, models.PluginEmail.String(), meta.ID)
	assert.NotEmpty(t, meta.Version)
	assert.NotEmpty(t, meta.Description)
}

func TestEmailPlugin_Config(t *testing.T) {
	config := emailtypes.EmailPluginConfig{
		Enabled:     true,
		Provider:    emailtypes.ProviderSMTP,
		FromAddress: "noreply@example.com",
	}

	plugin := New(config)
	got, ok := plugin.Config().(*emailtypes.EmailPluginConfig)
	require.True(t, ok, "Config() = %T, want *types.EmailPluginConfig", plugin.Config())
	assert.Same(t, plugin.PluginConfig, got)
	assert.Equal(t, config.Provider, got.Provider)
	assert.Equal(t, config.FromAddress, got.FromAddress)
}

func TestEmailPlugin_Close(t *testing.T) {
	assert.NoError(t, New(emailtypes.EmailPluginConfig{}).Close())
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
			name:           "rejects an unsupported config provider",
			configProvider: "sendgrid",
			wantErr:        true,
			errContains:    "unsupported email provider",
		},
		{
			name:        "errors when no provider is configured at all",
			wantErr:     true,
			errContains: constants.EnvEmailProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEmailEnv(t)
			setProviderCredentials(t)
			t.Setenv(constants.EnvEmailProvider, tt.envProvider)

			plugin := New(emailtypes.EmailPluginConfig{
				Enabled:     true,
				Provider:    tt.configProvider,
				FromAddress: "noreply@example.com",
			})

			err := plugin.Init(newTestPluginContext())

			if tt.wantErr {
				require.Error(t, err, "Init() expected an error, provider = %q", plugin.PluginConfig.Provider)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, plugin.PluginConfig.Provider)
		})
	}
}

func TestEmailPlugin_Init_NormalisesFallbackProvider(t *testing.T) {
	isolateEmailEnv(t)
	setProviderCredentials(t)
	t.Setenv(constants.EnvEmailProvider, "smtp")

	plugin := New(emailtypes.EmailPluginConfig{
		Enabled:          true,
		Provider:         emailtypes.ProviderResend,
		FallbackProvider: "SMTP",
		FromAddress:      "noreply@example.com",
	})

	require.NoError(t, plugin.Init(newTestPluginContext()))
	assert.Equal(t, emailtypes.ProviderSMTP, plugin.PluginConfig.FallbackProvider)

	// The env var promoted SMTP to primary, so it must not also be the fallback.
	assert.Nil(t, plugin.EmailService.fallbackProvider)
}

func TestEmailPlugin_Init_RejectsInvalidFallbackProvider(t *testing.T) {
	isolateEmailEnv(t)
	setProviderCredentials(t)

	plugin := New(emailtypes.EmailPluginConfig{
		Enabled:          true,
		Provider:         emailtypes.ProviderSMTP,
		FallbackProvider: "mailgun",
		FromAddress:      "noreply@example.com",
	})

	err := plugin.Init(newTestPluginContext())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid fallback email provider")
}

func TestEmailPlugin_Init_FromAddress(t *testing.T) {
	tests := []struct {
		name        string
		envFrom     string
		configFrom  string
		wantErr     string
		wantAddress string
	}{
		{
			name:        "uses the configured from address",
			configFrom:  "config@example.com",
			wantAddress: "config@example.com",
		},
		{
			name:        "env var overrides the configured from address",
			envFrom:     "override@example.com",
			configFrom:  "config@example.com",
			wantAddress: "override@example.com",
		},
		{
			name:        "env var supplies the from address when config omits it",
			envFrom:     "env@example.com",
			wantErr:     "",
			wantAddress: "env@example.com",
		},
		{
			name:    "errors when no from address is configured",
			wantErr: constants.EnvEmailFrom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEmailEnv(t)
			setProviderCredentials(t)
			t.Setenv(constants.EnvEmailProvider, "smtp")
			t.Setenv(constants.EnvEmailFrom, tt.envFrom)

			plugin := New(emailtypes.EmailPluginConfig{
				Enabled:     true,
				Provider:    emailtypes.ProviderSMTP,
				FromAddress: tt.configFrom,
			})

			err := plugin.Init(newTestPluginContext())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAddress, plugin.PluginConfig.FromAddress)
		})
	}
}

func TestEmailPlugin_Init_ConfigLoadFailureKeepsDefaults(t *testing.T) {
	isolateEmailEnv(t)
	setProviderCredentials(t)
	t.Setenv(constants.EnvEmailProvider, "smtp")

	logger := &internaltests.LevelRecordingLogger{}
	plugin := New(emailtypes.EmailPluginConfig{
		Enabled:     true,
		FromAddress: "noreply@example.com",
	})
	ctx := &models.PluginContext{
		Logger:          logger,
		ServiceRegistry: internalplugins.NewServiceRegistry(),
		GetConfig: func() *models.Config {
			return &models.Config{
				Plugins: models.PluginsConfig{models.PluginEmail.String(): "not-a-map"},
			}
		},
	}

	require.NoError(t, plugin.Init(ctx))
	assert.Equal(t, emailtypes.ProviderSMTP, plugin.PluginConfig.Provider)

	records := logger.Records()
	require.NotEmpty(t, records)
	assert.Equal(t, "warn", records[0].Level)
	assert.Contains(t, records[0].Message, "failed to load email plugin config")
}

func TestEmailPlugin_Init_ProviderInitializationFailure(t *testing.T) {
	tests := []struct {
		name        string
		envProvider string
		wantErr     string
	}{
		{
			name:        "wraps SMTP construction errors",
			envProvider: "smtp",
			wantErr:     "failed to initialize SMTP provider",
		},
		{
			name:        "wraps Resend construction errors",
			envProvider: "resend",
			wantErr:     "failed to initialize Resend provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEmailEnv(t)
			t.Setenv(constants.EnvEmailProvider, tt.envProvider)

			plugin := New(emailtypes.EmailPluginConfig{
				Enabled:     true,
				FromAddress: "noreply@example.com",
			})

			err := plugin.Init(newTestPluginContext())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestEmailPlugin_Init_FallbackProvider(t *testing.T) {
	primaryErr := "primary provider should be resend"

	tests := []struct {
		name            string
		fallback        emailtypes.EmailProviderType
		withSMTPCreds   bool
		wantFallback    bool
		wantFallbackTyp any
	}{
		{
			name:            "initialises a different fallback provider",
			fallback:        emailtypes.ProviderSMTP,
			withSMTPCreds:   true,
			wantFallback:    true,
			wantFallbackTyp: &providers.SMTPProvider{},
		},
		{
			name:         "skips the fallback when it matches the primary",
			fallback:     emailtypes.ProviderResend,
			wantFallback: false,
		},
		{
			name:         "skips the fallback when it fails to initialise",
			fallback:     emailtypes.ProviderSMTP,
			wantFallback: false,
		},
		{
			name:         "skips the fallback when none is configured",
			wantFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEmailEnv(t)
			t.Setenv(constants.EnvEmailProvider, "resend")
			t.Setenv(constants.EnvResendApiKey, "re_test_key")
			if tt.withSMTPCreds {
				t.Setenv(constants.EnvSMTPHost, "smtp.example.com")
				t.Setenv(constants.EnvSMTPPort, "587")
			}

			plugin := New(emailtypes.EmailPluginConfig{
				Enabled:          true,
				FallbackProvider: tt.fallback,
				FromAddress:      "noreply@example.com",
			})

			err := plugin.Init(newTestPluginContext())
			require.NoError(t, err)
			assert.Equal(t, emailtypes.ProviderResend, plugin.PluginConfig.Provider, primaryErr)

			if !tt.wantFallback {
				assert.Nil(t, plugin.EmailService.fallbackProvider)
				return
			}

			require.NotNil(t, plugin.EmailService.fallbackProvider)
			assert.IsType(t, tt.wantFallbackTyp, plugin.EmailService.fallbackProvider)
		})
	}
}

func TestEmailPlugin_Init_RegistersMailerAdapter(t *testing.T) {
	isolateEmailEnv(t)
	setProviderCredentials(t)
	t.Setenv(constants.EnvEmailProvider, "smtp")

	registry := internalplugins.NewServiceRegistry()
	ctx := &models.PluginContext{
		Logger:          &internaltests.MockLogger{},
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return &models.Config{} },
	}

	plugin := New(emailtypes.EmailPluginConfig{
		Enabled:     true,
		FromAddress: "noreply@example.com",
	})
	require.NoError(t, plugin.Init(ctx))

	registered := registry.Get(models.ServiceMailer.String())
	require.NotNil(t, registered)

	adapter, ok := registered.(*MailerServiceAdapter)
	require.True(t, ok, "registered service = %T, want *MailerServiceAdapter", registered)
	assert.Same(t, plugin.EmailService, adapter.emailService)
}

func TestEmailPlugin_InitializeProvider(t *testing.T) {
	tests := []struct {
		name      string
		provider  emailtypes.EmailProviderType
		envHost   string
		envPort   string
		envAPIKey string
		wantType  any
		wantErr   string
	}{
		{
			name:     "initialises the SMTP provider",
			provider: emailtypes.ProviderSMTP,
			envHost:  "smtp.example.com",
			envPort:  "587",
			wantType: &providers.SMTPProvider{},
		},
		{
			name:      "initialises the Resend provider",
			provider:  emailtypes.ProviderResend,
			envAPIKey: "re_test_key",
			wantType:  &providers.ResendProvider{},
		},
		{
			name:     "rejects an unsupported provider",
			provider: emailtypes.EmailProviderType("mailgun"),
			wantErr:  "unsupported email provider",
		},
		{
			name:     "wraps SMTP construction errors",
			provider: emailtypes.ProviderSMTP,
			wantErr:  "failed to initialize SMTP provider",
		},
		{
			name:     "wraps Resend construction errors",
			provider: emailtypes.ProviderResend,
			wantErr:  "failed to initialize Resend provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateEmailEnv(t)
			t.Setenv(constants.EnvSMTPHost, tt.envHost)
			t.Setenv(constants.EnvSMTPPort, tt.envPort)
			t.Setenv(constants.EnvResendApiKey, tt.envAPIKey)

			plugin := New(emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"})
			plugin.Logger = &internaltests.MockLogger{}

			got, err := plugin.initializeProvider(tt.provider)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.IsType(t, tt.wantType, got)
		})
	}
}

func TestEmailPlugin_Init_ErrorMentionsSupportedProviders(t *testing.T) {
	isolateEmailEnv(t)

	plugin := New(emailtypes.EmailPluginConfig{
		Enabled:     true,
		FromAddress: "noreply@example.com",
	})

	err := plugin.Init(newTestPluginContext())
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), emailtypes.SupportedEmailProvidersLabel()))
}
