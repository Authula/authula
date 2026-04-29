package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
	rootservices "github.com/Authula/authula/services"
)

type verifyEmailUseCase struct {
	PluginConfig        types.EmailPasswordPluginConfig
	Logger              models.Logger
	UserService         rootservices.UserService
	AccountService      rootservices.AccountService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
	EventBus            models.EventBus
}

func NewVerifyEmailUseCase(
	pluginConfig types.EmailPasswordPluginConfig,
	logger models.Logger,
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	verificationService rootservices.VerificationService,
	tokenService rootservices.TokenService,
	mailerService rootservices.MailerService,
	eventBus models.EventBus,
) VerifyEmailUseCase {
	return &verifyEmailUseCase{PluginConfig: pluginConfig, Logger: logger, UserService: userService, AccountService: accountService, VerificationService: verificationService, TokenService: tokenService, MailerService: mailerService, EventBus: eventBus}
}

func (uc *verifyEmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) (models.VerificationType, error) {
	hashedToken := uc.TokenService.Hash(tokenStr)

	verification, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return "", err
	}

	if verification == nil || verification.ExpiresAt.Before(time.Now()) {
		return "", constants.ErrInvalidOrExpiredToken
	}

	if verification.UserID == nil {
		return "", constants.ErrUserNotFound
	}

	user, err := uc.UserService.GetByID(ctx, *verification.UserID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", constants.ErrUserNotFound
	}

	switch verification.Type {
	case models.TypeEmailVerification:
		if err := uc.handleEmailVerification(ctx, user, verification.ID); err != nil {
			return "", err
		}
	case models.TypePasswordResetRequest, models.TypeEmailResetRequest:
		if err := uc.handleEmailChangeVerificationEmail(ctx, *verification.UserID, tokenStr, verification.Identifier); err != nil {
			return "", err
		}
	default:
		return "", constants.ErrInvalidEmailVerificationType
	}

	return verification.Type, nil
}

func (uc *verifyEmailUseCase) handleEmailVerification(ctx context.Context, user *models.User, tokenID string) error {
	user.EmailVerified = true
	if _, err := uc.UserService.Update(ctx, user); err != nil {
		return err
	}

	if err := uc.VerificationService.Delete(ctx, tokenID); err != nil {
		return err
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		return err
	}

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

	return nil
}

func (uc *verifyEmailUseCase) handleEmailChangeVerificationEmail(
	ctx context.Context,
	userID string,
	tokenValue string,
	newEmail string,
) error {
	reqCtx, _ := models.GetRequestContext(ctx)

	if newEmail == "" {
		return fmt.Errorf("new email cannot be empty")
	}

	hashedToken := uc.TokenService.Hash(tokenValue)
	verification, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return err
	}

	if verification == nil || verification.Type != models.TypeEmailResetRequest || verification.ExpiresAt.Before(time.Now()) {
		return constants.ErrInvalidOrExpiredToken
	}

	if verification.Identifier != newEmail {
		return constants.ErrInvalidEmailMatch
	}

	user, err := uc.UserService.GetByID(ctx, *verification.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return constants.ErrUserNotFound
	}
	if user.ID != userID {
		return constants.ErrUserNotAuthorized
	}

	existing, err := uc.UserService.GetByEmail(ctx, newEmail)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != user.ID {
		return constants.ErrEmailAlreadyExists
	}

	account, err := uc.AccountService.GetByUserIDAndProvider(ctx, user.ID, models.AuthProviderEmail.String())
	if err != nil {
		return err
	}
	if account == nil {
		return constants.ErrAccountNotFound
	}

	oldEmail := user.Email
	user.Email = newEmail
	user.EmailVerified = true
	if _, err := uc.UserService.Update(ctx, user); err != nil {
		return err
	}

	account.AccountID = newEmail
	if _, err := uc.AccountService.Update(ctx, account); err != nil {
		return err
	}

	if err := uc.VerificationService.Delete(ctx, verification.ID); err != nil {
		return err
	}

	uc.publishEmailChangedEvent(user, oldEmail, newEmail)

	sendChangedEmailToOldEmailCallbackHandled := false
	sendChangedEmailToNewEmailCallbackHandled := false

	if uc.PluginConfig.SendChangedEmailToOldEmail != nil {
		err := uc.PluginConfig.SendChangedEmailToOldEmail(types.SendChangedEmailToOldEmailParams{
			User:  *user,
			Email: oldEmail,
		}, reqCtx)

		if err != nil {
			uc.Logger.Error("failed to send changed email to old email via plugin callback", "err", err.Error())
		} else {
			sendChangedEmailToOldEmailCallbackHandled = true
		}
	}

	if uc.PluginConfig.SendChangedEmailToNewEmail != nil {
		err := uc.PluginConfig.SendChangedEmailToNewEmail(types.SendChangedEmailToNewEmailParams{
			User:  *user,
			Email: newEmail,
		}, reqCtx)

		if err != nil {
			uc.Logger.Error("failed to send changed email to new email via plugin callback", "err", err.Error())
		} else {
			sendChangedEmailToNewEmailCallbackHandled = true
		}
	}

	if !sendChangedEmailToOldEmailCallbackHandled && !sendChangedEmailToNewEmailCallbackHandled && uc.MailerService != nil {
		go func() {
			detachedCtx := context.WithoutCancel(ctx)
			taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
			defer cancel()

			if err := uc.sendChangedEmailEmails(taskCtx, oldEmail, newEmail); err != nil {
				uc.Logger.Error("failed to send changed email emails via built-in email service", "err", err)
			}
		}()
	}

	return nil
}

func (uc *verifyEmailUseCase) sendChangedEmailEmails(ctx context.Context, oldEmail string, newEmail string) error {
	subject := "Your email has been changed"
	textBody := fmt.Sprintf("Your account email has been changed from %s to %s. If you did not perform this action, please contact support.", oldEmail, newEmail)

	if err := uc.MailerService.SendEmail(ctx, oldEmail, subject, textBody, getHtmlBody(oldEmail, oldEmail, newEmail)); err != nil {
		uc.Logger.Error("failed to send email to old address via built-in email service", "err", err)
	}

	if err := uc.MailerService.SendEmail(ctx, newEmail, subject, textBody, getHtmlBody(newEmail, oldEmail, newEmail)); err != nil {
		uc.Logger.Error("failed to send email to new address via built-in email service", "err", err)
	}

	return nil
}

func getHtmlBody(userEmail string, oldEmail string, newEmail string) string {
	return fmt.Sprintf(
		`<div>
			<p>Hello %s,</p>
			<p>Your account email has been changed from %s to %s. If you did not perform this action, please contact support immediately.</p>
		</div>`,
		userEmail,
		oldEmail,
		newEmail,
	)
}

func (uc *verifyEmailUseCase) publishEmailChangedEvent(user *models.User, oldEmail string, newEmail string) {
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
			Type:      constants.EventUserEmailChanged,
			Payload:   userJson,
			Metadata:  map[string]string{"old_email": oldEmail, "new_email": newEmail},
			Timestamp: time.Now().UTC(),
		},
	)
}
