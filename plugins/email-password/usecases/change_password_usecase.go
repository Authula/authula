package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type ChangePasswordUseCase struct {
	Logger              models.Logger
	PluginConfig        types.EmailPasswordPluginConfig
	UserService         rootservices.UserService
	AccountService      rootservices.AccountService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	PasswordService     rootservices.PasswordService
	MailerService       rootservices.MailerService
	EventBus            models.EventBus
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

	hashedToken := uc.TokenService.Hash(tokenValue)
	verification, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return err
	}

	if verification == nil ||
		verification.Type != models.TypePasswordResetRequest ||
		verification.ExpiresAt.Before(time.Now()) {
		return constants.ErrInvalidOrExpiredToken
	}

	user, err := uc.UserService.GetByID(ctx, *verification.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return constants.ErrUserNotFound
	}

	account, err := uc.AccountService.GetByUserIDAndProvider(ctx, *verification.UserID, models.AuthProviderEmail.String())
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

	if err := uc.VerificationService.Delete(ctx, verification.ID); err != nil {
		return err
	}

	go func() {
		detachedCtx := context.WithoutCancel(ctx)
		taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
		defer cancel()

		if err := uc.sendChangedPasswordEmail(taskCtx, user); err != nil {
			uc.Logger.Error("failed to send changed password email", "err", err)
		}
	}()

	uc.publishChangedPasswordEvent(user)

	return nil
}

func (uc *ChangePasswordUseCase) sendChangedPasswordEmail(ctx context.Context, user *models.User) error {
	subject := "Your password has been changed"
	textBody := "Your password has been successfully changed. If you did not perform this action, please reset your password immediately by requesting a password reset."
	htmlBody := fmt.Sprintf(
		`<div>
			<p>Hello %s,</p>
			<p>Your password has been successfully changed. If you did not perform this action, please reset your password immediately by requesting a password reset.</p>
		</div>`,
		user.Email,
	)
	return uc.MailerService.SendEmail(ctx, user.Email, subject, textBody, htmlBody)
}

func (uc *ChangePasswordUseCase) publishChangedPasswordEvent(user *models.User) {
	userJson, err := json.Marshal(user)
	if err != nil {
		uc.Logger.Error(err.Error())
		return
	}

	util.PublishEventAsync(
		uc.EventBus,
		uc.Logger,
		models.Event{
			ID:        util.GenerateUUID(),
			Type:      constants.EventUserChangedPassword,
			Payload:   userJson,
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}
