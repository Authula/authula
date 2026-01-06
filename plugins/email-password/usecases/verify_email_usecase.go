package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type VerifyEmailUseCase struct {
	Logger              models.Logger
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	EventBus            models.EventBus
}

func (uc *VerifyEmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) error {
	hashedToken := uc.TokenService.Hash(tokenStr)

	token, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return err
	}

	if token == nil || token.Type != models.TypeEmailVerification || token.ExpiresAt.Before(time.Now()) {
		return constants.ErrInvalidOrExpiredToken
	}

	if token.UserID == nil {
		return constants.ErrInvalidOrExpiredToken
	}

	user, err := uc.UserService.GetByID(ctx, *token.UserID)
	if err != nil {
		return err
	}

	if user == nil {
		return constants.ErrUserNotFound
	}

	user.EmailVerified = true
	if _, err := uc.UserService.Update(ctx, user); err != nil {
		return err
	}

	if err := uc.VerificationService.Delete(ctx, token.ID); err != nil {
		return err
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		uc.Logger.Error(err.Error())
	} else {
		util.PublishEventAsync(
			uc.EventBus,
			uc.Logger,
			models.Event{
				ID:        util.GenerateUUID(),
				Type:      constants.EventUserEmailVerified,
				Payload:   userJson,
				Metadata:  nil,
				Timestamp: time.Now().UTC(),
			},
		)
	}

	return nil
}
