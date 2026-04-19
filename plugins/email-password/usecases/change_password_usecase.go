package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	internalutil "github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
	rootservices "github.com/Authula/authula/services"
)

type changePasswordUseCase struct {
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

func NewChangePasswordUseCase(
	logger models.Logger,
	pluginConfig types.EmailPasswordPluginConfig,
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	verificationService rootservices.VerificationService,
	tokenService rootservices.TokenService,
	passwordService rootservices.PasswordService,
	mailerService rootservices.MailerService,
	eventBus models.EventBus,
) ChangePasswordUseCase {
	return &changePasswordUseCase{Logger: logger, PluginConfig: pluginConfig, UserService: userService, AccountService: accountService, VerificationService: verificationService, TokenService: tokenService, PasswordService: passwordService, MailerService: mailerService, EventBus: eventBus}
}

func (uc *changePasswordUseCase) ChangePassword(
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

func (uc *changePasswordUseCase) sendChangedPasswordEmail(ctx context.Context, user *models.User) error {
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

func (uc *changePasswordUseCase) publishChangedPasswordEvent(user *models.User) {
	userJson, err := json.Marshal(user)
	if err != nil {
		uc.Logger.Error(err.Error())
		return
	}

	internalutil.PublishEventAsync(
		uc.EventBus,
		uc.Logger,
		models.Event{
			ID:        internalutil.GenerateUUID(),
			Type:      constants.EventUserChangedPassword,
			Payload:   userJson,
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}
