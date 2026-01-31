package providers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/wneessen/go-mail"

	"github.com/GoBetterAuth/go-better-auth/v2/env"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	emailtypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/email/types"
)

type SMTPProvider struct {
	config *emailtypes.EmailPluginConfig
	logger models.Logger

	host string
	port int
	user string
	pass string
}

func NewSMTPProvider(
	config *emailtypes.EmailPluginConfig,
	logger models.Logger,
) (*SMTPProvider, error) {

	host := strings.TrimSpace(os.Getenv(env.EnvSMTPHost))
	if host == "" {
		return nil, fmt.Errorf("%s environment variable is not set", env.EnvSMTPHost)
	}

	portStr := strings.TrimSpace(os.Getenv(env.EnvSMTPPort))
	if portStr == "" {
		return nil, fmt.Errorf("%s environment variable is not set", env.EnvSMTPPort)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid integer: %w", env.EnvSMTPPort, err)
	}

	user := strings.TrimSpace(os.Getenv(env.EnvSMTPUser))
	pass := strings.TrimSpace(os.Getenv(env.EnvSMTPPass))

	return &SMTPProvider{
		config: config,
		logger: logger,
		host:   host,
		port:   port,
		user:   user,
		pass:   pass,
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

	// Plain text is required
	msg.SetBodyString(mail.TypeTextPlain, text)

	// HTML is optional
	if html != "" {
		msg.AddAlternativeString(mail.TypeTextHTML, html)
	}

	opts := []mail.Option{
		mail.WithPort(s.port),
	}

	opts = append(opts,
		mail.WithUsername(s.user),
		mail.WithPassword(s.pass),
		mail.WithTLSPolicy(mail.TLSOpportunistic),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
	)

	client, err := mail.NewClient(s.host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
