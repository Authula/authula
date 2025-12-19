package sendverificationemail

import "context"

type SendEmailVerificationUseCase interface {
	SendEmailVerification(ctx context.Context, userID string, callbackURL *string) error
}
