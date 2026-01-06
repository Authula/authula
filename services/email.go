package services

import "context"

type MailerService interface {
	SendEmail(ctx context.Context, to string, subject string, text string, html string) error
}
