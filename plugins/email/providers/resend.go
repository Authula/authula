package providers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/resend/resend-go/v3"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email/constants"
	emailtypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/email/types"
)

type ResendProvider struct {
	config *emailtypes.EmailPluginConfig
	logger models.Logger
	client *resend.Client
}

func NewResendProvider(
	config *emailtypes.EmailPluginConfig,
	logger models.Logger,
) (*ResendProvider, error) {
	apiKey := strings.TrimSpace(os.Getenv(constants.EnvResendApiKey))
	if apiKey == "" {
		if config.Resend != nil && config.Resend.ApiKey != "" {
			apiKey = config.Resend.ApiKey
		} else {
			logger.Error("Resend API key is not set in environment variable or config")
			return nil, fmt.Errorf("%s environment variable is not set", constants.EnvResendApiKey)
		}
	}

	client := resend.NewClient(apiKey)

	return &ResendProvider{
		config: config,
		logger: logger,
		client: client,
	}, nil
}

func (r *ResendProvider) SendEmail(
	ctx context.Context,
	to string,
	subject string,
	text string,
	html string,
) error {
	if text == "" && html == "" {
		return fmt.Errorf("email must have at least a text or html body")
	}

	params := &resend.SendEmailRequest{
		To:      []string{to},
		From:    r.config.FromAddress,
		Subject: subject,
		Text:    text,
		Html:    html,
	}

	sent, err := r.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		r.logger.Error("failed to send email via Resend", map[string]any{
			"provider": "resend",
			"to":       to,
			"subject":  subject,
			"error":    err.Error(),
		})
		return fmt.Errorf("resend send failed: %w", err)
	}

	if sent == nil || sent.Id == "" {
		return fmt.Errorf("resend send failed: empty response")
	}

	return nil
}
