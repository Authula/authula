package signup

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type SignUpUseCase interface {
	SignUpWithEmailAndPassword(ctx context.Context, name string, email string, password string, callbackURL *string) (*models.SignUpResult, error)
}
