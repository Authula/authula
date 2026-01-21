package providers

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/wneessen/go-mail"

	"github.com/GoBetterAuth/go-better-auth/env"
	"github.com/GoBetterAuth/go-better-auth/models"
	emailtypes "github.com/GoBetterAuth/go-better-auth/plugins/email/types"
)

type SMTPProvider struct {
	config *emailtypes.EmailPluginConfig
	logger models.Logger
	host   string
	port   int
	user   string
	pass   string
}

func NewSMTPProvider(config *emailtypes.EmailPluginConfig, logger models.Logger) (*SMTPProvider, error) {
	host := os.Getenv(env.EnvSMTPHost)
	if host == "" {
		return nil, fmt.Errorf("%s environment variable is not set", env.EnvSMTPHost)
	}

	portStr := os.Getenv(env.EnvSMTPPort)
	if portStr == "" {
		return nil, fmt.Errorf("%s environment variable is not set", env.EnvSMTPPort)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid integer: %w", env.EnvSMTPPort, err)
	}

	user := os.Getenv(env.EnvSMTPUser)
	if user == "" {
		return nil, fmt.Errorf("%s environment variable is not set", env.EnvSMTPUser)
	}

	pass := os.Getenv(env.EnvSMTPPass)
	if pass == "" {
		return nil, fmt.Errorf("%s environment variable is not set", env.EnvSMTPPass)
	}

	return &SMTPProvider{
		config: config,
		logger: logger,
		host:   host,
		port:   port,
		user:   user,
		pass:   pass,
	}, nil
}

func (s *SMTPProvider) SendEmail(ctx context.Context, to string, subject string, text string, html string) error {
	msg := mail.NewMsg()

	if err := msg.From(s.config.FromAddress); err != nil {
		return err
	}
	if err := msg.To(to); err != nil {
		return err
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, text)
	msg.SetBodyString(mail.TypeTextHTML, html)

	opts := []mail.Option{
		mail.WithPort(s.port),
	}

	// isLocal := s.host == "localhost" || s.host == "127.0.0.1" || s.host == "0.0.0.0" || s.host == "host.docker.internal"

	// if isLocal {
	// 	opts = append(opts,
	// 		mail.WithTLSPolicy(mail.NoTLS), // Local servers like MailDev/Nodemailer usually don't have SSL
	// 	)
	// } else {
	// 	opts = append(opts,
	// 		mail.WithUsername(s.user),
	// 		mail.WithPassword(s.pass),
	// 		mail.WithTLSPolicy(mail.TLSMandatory), // Ensure encrypted connection for production
	// 		mail.WithSMTPAuth(mail.SMTPAuthLogin), // Standard for most cloud providers
	// 	)
	// }

	opts = append(opts,
		mail.WithTLSPolicy(mail.NoTLS), // Local servers like MailDev/Nodemailer usually don't have SSL
	)
	client, err := mail.NewClient(s.host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	return client.DialAndSendWithContext(ctx, msg)
}
