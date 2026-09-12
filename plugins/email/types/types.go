package types

import (
	"fmt"
	"slices"
	"strings"
)

// TemplateData contains all data needed for email template rendering
type TemplateData struct {
	UserName string            `json:"user_name"`
	Email    string            `json:"email"`
	Token    string            `json:"token"`
	URL      string            `json:"url"`
	AppName  string            `json:"app_name"`
	Extra    map[string]string `json:"extra"`
}

type EmailProviderType string

const (
	ProviderSMTP   EmailProviderType = "smtp"
	ProviderResend EmailProviderType = "resend"
)

func (e EmailProviderType) String() string {
	return string(e)
}

func SupportedEmailProviders() []EmailProviderType {
	return []EmailProviderType{ProviderSMTP, ProviderResend}
}

func ParseEmailProviderType(raw string) (EmailProviderType, error) {
	provider := EmailProviderType(strings.ToLower(strings.TrimSpace(raw)))

	if slices.Contains(SupportedEmailProviders(), provider) {
		return provider, nil
	}

	return "", fmt.Errorf("unsupported email provider %q, supported providers are: %s", raw, SupportedEmailProvidersLabel())
}

func SupportedEmailProvidersLabel() string {
	supported := SupportedEmailProviders()
	labels := make([]string, 0, len(supported))
	for _, provider := range supported {
		labels = append(labels, provider.String())
	}

	return strings.Join(labels, ", ")
}
