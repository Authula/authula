package verifyemail

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type VerifyEmailUseCase interface {
	VerifyEmail(ctx context.Context, rawToken string) (*models.VerifyEmailResult, error)
}
