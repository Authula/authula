package usecases

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type ChangePasswordUseCase struct {
	PluginConfig        types.EmailPasswordPluginConfig
	AccountService      rootservices.AccountService
	VerificationService rootservices.VerificationService
	PasswordService     rootservices.PasswordService
}

func (uc *ChangePasswordUseCase) ChangePassword(
	ctx context.Context,
	tokenValue string,
	newPassword string,
) error {
	if len(newPassword) < uc.PluginConfig.MinPasswordLength ||
		len(newPassword) > uc.PluginConfig.MaxPasswordLength {
		return constants.ErrInvalidPasswordLength
	}

	token, err := uc.VerificationService.GetByToken(ctx, tokenValue)
	if err != nil {
		return err
	}

	if token == nil ||
		token.Type != models.TypePasswordResetRequest ||
		token.ExpiresAt.Before(time.Now()) {
		return constants.ErrInvalidOrExpiredToken
	}

	account, err := uc.AccountService.GetByUserIDAndProvider(ctx, *token.UserID, "email_password")
	if err != nil {
		return err
	}

	if account == nil {
		return constants.ErrAccountNotFound
	}

	hash, err := uc.PasswordService.Hash(newPassword)
	if err != nil {
		return err
	}

	account.Password = &hash
	if _, err := uc.AccountService.Update(ctx, account); err != nil {
		return err
	}

	if err := uc.VerificationService.Delete(ctx, token.ID); err != nil {
		return err
	}

	return nil
}
