package email

import (
	"context"
	"fmt"
	"html/template"
	texttemplate "text/template"

	"github.com/GoBetterAuth/go-better-auth/models"
	emailtypes "github.com/GoBetterAuth/go-better-auth/plugins/email/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type EmailService struct {
	logger           models.Logger
	config           *emailtypes.EmailPluginConfig
	primaryProvider  rootservices.MailerService
	fallbackProvider rootservices.MailerService
	htmlTpls         map[string]*template.Template
	textTpls         map[string]*texttemplate.Template
}

func NewEmailService(
	logger models.Logger,
	config *emailtypes.EmailPluginConfig,
	primary rootservices.MailerService,
	fallback rootservices.MailerService,
) (*EmailService, error) {
	service := &EmailService{
		logger:           logger,
		config:           config,
		primaryProvider:  primary,
		fallbackProvider: fallback,
		htmlTpls:         make(map[string]*template.Template),
		textTpls:         make(map[string]*texttemplate.Template),
	}

	return service, nil
}

// SendEmail sends an email via the primary provider with optional fallback
func (s *EmailService) SendEmail(ctx context.Context, to, subject, text, html string) error {
	// Try primary provider
	err := s.primaryProvider.SendEmail(ctx, to, subject, text, html)
	if err == nil {
		return nil
	}

	s.logger.Warn(fmt.Sprintf("primary email provider failed: %v", err), map[string]any{
		"to":      to,
		"subject": subject,
	})

	// Try fallback provider if configured
	if s.fallbackProvider != nil {
		fallbackErr := s.fallbackProvider.SendEmail(ctx, to, subject, text, html)
		if fallbackErr == nil {
			s.logger.Info("fallback email provider succeeded", map[string]any{
				"to": to,
			})
			return nil
		}

		s.logger.Error(fmt.Sprintf("fallback email provider also failed: %v", fallbackErr), map[string]any{
			"to":      to,
			"subject": subject,
		})
	}

	return err
}
