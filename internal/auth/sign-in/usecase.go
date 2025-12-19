package signin

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type SignInUseCase interface {
	SignInWithEmailAndPassword(ctx context.Context, email string, password string, callbackURL *string) (*models.SignInResult, error)
}
