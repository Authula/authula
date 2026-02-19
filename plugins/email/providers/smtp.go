package providers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/wneessen/go-mail"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email/types"
)

type SMTPProvider struct {
	config    *types.EmailPluginConfig
	logger    models.Logger
	host      string
	port      int
	user      string
	pass      string
	tlsPolicy mail.TLSPolicy
}

func NewSMTPProvider(
	config *types.EmailPluginConfig,
	logger models.Logger,
) (*SMTPProvider, error) {
	host := strings.TrimSpace(os.Getenv(constants.EnvSMTPHost))
	if host == "" {
		if config.SMTP == nil || strings.TrimSpace(config.SMTP.Host) == "" {
			return nil, fmt.Errorf("%s environment variable is not set and no SMTP host provided in config", constants.EnvSMTPHost)
		}
		host = strings.TrimSpace(config.SMTP.Host)
	}

	portStr := strings.TrimSpace(os.Getenv(constants.EnvSMTPPort))
	if portStr == "" {
		if config.SMTP == nil || config.SMTP.Port == 0 {
			return nil, fmt.Errorf("%s environment variable is not set and no SMTP port provided in config", constants.EnvSMTPPort)
		}
		portStr = strconv.Itoa(config.SMTP.Port)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid integer: %w", constants.EnvSMTPPort, err)
	}

	tlsMode := strings.ToLower(strings.TrimSpace(config.TLSMode.String()))
	var tlsPolicy mail.TLSPolicy
	switch tlsMode {
	case types.SMTPTLSModeOff.String():
		tlsPolicy = mail.NoTLS
	case types.SMTPTLSModeStartTLS.String(), "": // default
		tlsPolicy = mail.TLSOpportunistic
	case types.SMTPTLSModeTLS.String():
		tlsPolicy = mail.TLSMandatory
	default:
		return nil, fmt.Errorf("invalid SMTP_TLS_MODE: %q", tlsMode)
	}

	// If credentials are supplied, require TLS (prevent AUTH over plaintext).
	user := strings.TrimSpace(os.Getenv(constants.EnvSMTPUser))
	if user == "" {
		if config.SMTP != nil && strings.TrimSpace(config.SMTP.Username) != "" {
			user = strings.TrimSpace(config.SMTP.Username)
		}
	}
	pass := strings.TrimSpace(os.Getenv(constants.EnvSMTPPass))
	if pass == "" {
		if config.SMTP != nil && strings.TrimSpace(config.SMTP.Password) != "" {
			pass = strings.TrimSpace(config.SMTP.Password)
		}
	}
	if (user != "" || pass != "") && tlsPolicy == mail.NoTLS {
		return nil, fmt.Errorf(
			"SMTP credentials supplied but TLS is disabled; set tls_mode to %q or %q, or remove SMTP_USER/SMTP_PASS",
			types.SMTPTLSModeStartTLS.String(),
			types.SMTPTLSModeTLS.String(),
		)
	}

	return &SMTPProvider{
		config:    config,
		logger:    logger,
		host:      host,
		port:      port,
		user:      user,
		pass:      pass,
		tlsPolicy: tlsPolicy,
	}, nil
}

func (s *SMTPProvider) SendEmail(
	ctx context.Context,
	to string,
	subject string,
	text string,
	html string,
) error {
	msg := mail.NewMsg()

	if err := msg.From(s.config.FromAddress); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	if err := msg.To(to); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}

	msg.Subject(subject)

	msg.SetBodyString(mail.TypeTextPlain, text)

	if html != "" {
		msg.AddAlternativeString(mail.TypeTextHTML, html)
	}

	opts := []mail.Option{
		mail.WithPort(s.port),
		mail.WithTLSPolicy(s.tlsPolicy),
	}

	if s.user != "" || s.pass != "" {
		opts = append(opts,
			mail.WithUsername(s.user),
			mail.WithPassword(s.pass),
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
		)
	}

	client, err := mail.NewClient(s.host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
