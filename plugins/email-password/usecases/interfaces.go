package usecases

import (
	"context"
	"encoding/json"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
)

type UseCases struct {
	SignUpUseCase                SignUpUseCase
	SignInUseCase                SignInUseCase
	VerifyEmailUseCase           VerifyEmailUseCase
	SendEmailVerificationUseCase SendEmailVerificationUseCase
	RequestPasswordResetUseCase  RequestPasswordResetUseCase
	ChangePasswordUseCase        ChangePasswordUseCase
	RequestEmailChangeUseCase    RequestEmailChangeUseCase
}

type SignUpUseCase interface {
	SignUp(ctx context.Context, name string, email string, password string, image *string, metadata json.RawMessage, callbackURL *string, ipAddress *string, userAgent *string) (*types.SignUpResult, error)
}

type SignInUseCase interface {
	SignIn(ctx context.Context, email string, password string, callbackURL *string, ipAddress *string, userAgent *string) (*types.SignInResult, error)
	GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
}

type VerifyEmailUseCase interface {
	VerifyEmail(ctx context.Context, tokenStr string) (models.VerificationType, error)
}

type SendEmailVerificationUseCase interface {
	Send(ctx context.Context, userID string, callbackURL *string) error
}

type RequestPasswordResetUseCase interface {
	RequestReset(ctx context.Context, email string, callbackURL *string) error
}

type ChangePasswordUseCase interface {
	ChangePassword(ctx context.Context, tokenValue string, newPassword string) error
}

type RequestEmailChangeUseCase interface {
	RequestChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error
}
