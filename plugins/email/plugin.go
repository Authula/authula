package email

import (
	"context"
	"fmt"
	"os"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email/providers"
	emailtypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/email/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
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

	primaryProvider, err := p.initializeProvider(p.PluginConfig.Provider, true)
	if err != nil {
		return err
	}

	var fallbackProvider rootservices.MailerService
	if p.PluginConfig.FallbackProvider != "" && p.PluginConfig.FallbackProvider != p.PluginConfig.Provider {
		fallbackProvider, _ = p.initializeProvider(p.PluginConfig.FallbackProvider, false)
	}

	emailService, err := NewEmailService(p.Logger, p.PluginConfig, primaryProvider, fallbackProvider)
	if err != nil {
		return fmt.Errorf("failed to initialize email service: %w", err)
	}
	p.EmailService = emailService

	ctx.ServiceRegistry.Register(models.ServiceMailer.String(), NewMailerServiceAdapter(emailService))

	return nil
}

// initializeProvider creates a provider instance based on the provider type
// isPrimary indicates whether this is a primary provider (returns error) or fallback (logs warning)
func (p *EmailPlugin) initializeProvider(providerType emailtypes.EmailProviderType, isPrimary bool) (rootservices.MailerService, error) {
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
