package email

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email/constants"
	"github.com/Authula/authula/plugins/email/providers"
	emailtypes "github.com/Authula/authula/plugins/email/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type EmailPlugin struct {
	PluginConfig *emailtypes.EmailPluginConfig
	Logger       models.Logger
	ctx          *models.PluginContext
	EmailService *EmailService
}

func New(config emailtypes.EmailPluginConfig) *EmailPlugin {
	return &EmailPlugin{
		PluginConfig: &config,
	}
}

func (p *EmailPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginEmail.String(),
		Version:     "1.0.0",
		Description: "Email plugin with providers, template rendering, and tiered error handling.",
	}
}

func (p *EmailPlugin) Config() any {
	return p.PluginConfig
}

func (p *EmailPlugin) Init(ctx *models.PluginContext) error {
	p.Logger = ctx.Logger
	p.ctx = ctx
	globalConfig := ctx.GetConfig()

	if err := util.LoadPluginConfig(globalConfig, p.Metadata().ID, p.PluginConfig); err != nil {
		p.Logger.Warn("failed to load email plugin config, using defaults", map[string]any{
			"error": err.Error(),
		})
	}

	if emailFrom := os.Getenv(constants.EnvEmailFrom); emailFrom != "" {
		p.PluginConfig.FromAddress = emailFrom
	}

	if p.PluginConfig.FromAddress == "" {
		return fmt.Errorf("email plugin requires 'from_address' to be configured in %s env var or config", constants.EnvEmailFrom)
	}

	if err := p.resolveProviders(); err != nil {
		return err
	}

	primaryProvider, err := p.initializeProvider(p.PluginConfig.Provider)
	if err != nil {
		return err
	}

	var fallbackProvider rootservices.MailerService
	if p.PluginConfig.FallbackProvider != "" && p.PluginConfig.FallbackProvider != p.PluginConfig.Provider {
		fallbackProvider, _ = p.initializeProvider(p.PluginConfig.FallbackProvider)
	}

	emailService, err := NewEmailService(p.Logger, p.PluginConfig, primaryProvider, fallbackProvider)
	if err != nil {
		return fmt.Errorf("failed to initialize email service: %w", err)
	}
	p.EmailService = emailService

	ctx.ServiceRegistry.Register(models.ServiceMailer.String(), NewMailerServiceAdapter(emailService))

	return nil
}

func (p *EmailPlugin) resolveProviders() error {
	rawProvider := strings.TrimSpace(os.Getenv(constants.EnvEmailProvider))
	fromEnv := rawProvider != ""
	if !fromEnv {
		rawProvider = strings.TrimSpace(p.PluginConfig.Provider.String())
	}

	if rawProvider == "" {
		return fmt.Errorf(
			"email plugin requires 'provider' to be configured in %s env var or config, supported providers are: %s",
			constants.EnvEmailProvider,
			emailtypes.SupportedEmailProvidersLabel(),
		)
	}

	provider, err := emailtypes.ParseEmailProviderType(rawProvider)
	if err != nil {
		if fromEnv {
			return fmt.Errorf("invalid %s env var: %w", constants.EnvEmailProvider, err)
		}
		return err
	}
	p.PluginConfig.Provider = provider

	if rawFallback := strings.TrimSpace(p.PluginConfig.FallbackProvider.String()); rawFallback != "" {
		fallback, err := emailtypes.ParseEmailProviderType(rawFallback)
		if err != nil {
			return fmt.Errorf("invalid fallback email provider: %w", err)
		}
		p.PluginConfig.FallbackProvider = fallback
	}

	return nil
}

func (p *EmailPlugin) initializeProvider(providerType emailtypes.EmailProviderType) (rootservices.MailerService, error) {
	var provider rootservices.MailerService
	var err error

	switch providerType {
	case emailtypes.ProviderSMTP:
		provider, err = providers.NewSMTPProvider(p.PluginConfig, p.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SMTP provider: %w", err)
		}

	case emailtypes.ProviderResend:
		provider, err = providers.NewResendProvider(p.PluginConfig, p.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Resend provider: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported email provider: %s", providerType)
	}

	return provider, nil
}

func (p *EmailPlugin) Close() error {
	return nil
}

type MailerServiceAdapter struct {
	emailService *EmailService
}

func NewMailerServiceAdapter(emailService *EmailService) *MailerServiceAdapter {
	return &MailerServiceAdapter{
		emailService: emailService,
	}
}

func (a *MailerServiceAdapter) SendEmail(ctx context.Context, to string, subject string, text string, html string) error {
	return a.emailService.SendEmail(ctx, to, subject, text, html)
}
