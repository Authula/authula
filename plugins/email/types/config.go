package types

// EmailPluginConfig contains configuration for the email plugin
type EmailPluginConfig struct {
	Enabled bool `json:"enabled" toml:"enabled"`

	// Primary provider to use
	Provider EmailProviderType `json:"provider" toml:"provider"`

	// Optional fallback provider if primary fails
	FallbackProvider EmailProviderType `json:"fallback_provider" toml:"fallback_provider"`

	// FromAddress is the email address to send from
	FromAddress string `json:"from_address" toml:"from_address"`

	// TLSMode defines the TLS mode for SMTP provider
	TLSMode SMTPTLSMode `json:"tls_mode" toml:"tls_mode"`

	// SMTP configuration
	SMTP *SMTPConfig `json:"smtp" toml:"smtp"`

	// Resend configuration
	Resend *ResendConfig `json:"resend" toml:"resend"`
}

type SMTPTLSMode string

const (
	SMTPTLSModeOff      SMTPTLSMode = "off"
	SMTPTLSModeStartTLS SMTPTLSMode = "starttls"
	SMTPTLSModeTLS      SMTPTLSMode = "tls"
)

func (m SMTPTLSMode) String() string {
	return string(m)
}

type SMTPConfig struct {
	Host     string `json:"host" toml:"host"`
	Port     int    `json:"port" toml:"port"`
	Username string `json:"username" toml:"username"`
	Password string `json:"password" toml:"password"`
}

type ResendConfig struct {
	ApiKey string `json:"api_key" toml:"api_key"`
}
