package services

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type MailerServiceImpl struct {
	config *models.Config
}

func NewMailerServiceImpl(config *models.Config) *MailerServiceImpl {
	return &MailerServiceImpl{
		config: config,
	}
}

func (s *MailerServiceImpl) Send(ctx context.Context, to string, subject string, body string, htmlBody string) error {
	if s.config.Email.Provider != "smtp" {
		return fmt.Errorf("unsupported email provider: %s", s.config.Email.Provider)
	}

	if s.config.Email.SMTPHost == "" || s.config.Email.SMTPPort == 0 || s.config.Email.From == "" {
		return fmt.Errorf("invalid email configuration: missing required fields")
	}

	// Build the email message
	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n",
		s.config.Email.From,
		to,
		subject,
	)
	message := headers + "\r\n" + body

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.Email.SMTPHost, s.config.Email.SMTPPort)
	auth := smtp.PlainAuth("", s.config.Email.SMTPUser, s.config.Email.SMTPPass, s.config.Email.SMTPHost)

	err := smtp.SendMail(addr, auth, s.config.Email.From, []string{to}, []byte(message))
	if err != nil {
		s.config.Logger.Logger.Error("failed to send email via SMTP", "to", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.config.Logger.Logger.Info("email sent successfully", "to", to, "subject", subject)

	return nil
}
