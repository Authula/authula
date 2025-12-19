package emailchange

import "context"

type EmailChangeUseCase interface {
	EmailChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error
}
