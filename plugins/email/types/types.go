package types

// TemplateData contains all data needed for email template rendering
type TemplateData struct {
	UserName string            `json:"user_name"`
	Email    string            `json:"email"`
	Token    string            `json:"token"`
	URL      string            `json:"url"`
	AppName  string            `json:"app_name"`
	Extra    map[string]string `json:"extra"`
}

// EmailProviderType defines the email provider to use
type EmailProviderType string

const (
	ProviderSMTP   EmailProviderType = "smtp"
	ProviderResend EmailProviderType = "resend"
)

// EmailPluginConfig contains configuration for the email plugin
type EmailPluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`

	// Primary provider to use
	Provider EmailProviderType `json:"provider" toml:"provider"`

	// Optional fallback provider if primary fails
	FallbackProvider EmailProviderType `json:"fallback_provider" toml:"fallback_provider"`

	// FromAddress is the email address to send from
	FromAddress string `json:"from_address" toml:"from_address"`
}
