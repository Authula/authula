package resetpassword

import "context"

type ResetPasswordUseCase interface {
	ResetPassword(ctx context.Context, email string, callbackURL *string) error
}
