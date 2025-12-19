package changepassword

import (
	"context"
)

type ChangePasswordUseCase interface {
	ChangePassword(ctx context.Context, rawToken string, newPassword string) error
}
