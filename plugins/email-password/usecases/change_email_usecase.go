package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type ChangeEmailUseCase struct {
	Logger              models.Logger
	PluginConfig        types.EmailPasswordPluginConfig
	UserService         rootservices.UserService
	AccountService      rootservices.AccountService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
	EventBus            models.EventBus
}

func (uc *ChangeEmailUseCase) ChangeEmail(
	ctx context.Context,
	userID string,
	tokenValue string,
	newEmail string,
) error {
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
		return constants.ErrInvalidOrExpiredToken
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

	// Check duplicate
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

	go func() {
		detachedCtx := context.WithoutCancel(ctx)
		taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
		defer cancel()

		if err := uc.sendChangedEmailNotifications(taskCtx, user, oldEmail, newEmail); err != nil {
			uc.Logger.Error("failed to send changed email notifications", "err", err)
		}
	}()

	uc.publishEmailChangedEvent(user, oldEmail, newEmail)

	return nil
}

func (uc *ChangeEmailUseCase) sendChangedEmailNotifications(ctx context.Context, user *models.User, oldEmail, newEmail string) error {
	reqCtx, _ := models.GetRequestContext(ctx)
	if uc.PluginConfig.SendEmailChangeEmail != nil {
		if err := uc.PluginConfig.SendEmailChangeEmail(
			types.SendEmailChangeEmailParams{
				User:     *user,
				URL:      "",
				Token:    "",
				NewEmail: newEmail,
				OldEmail: oldEmail,
			},
			reqCtx,
		); err != nil {
			return err
		}
		return nil
	}

	subject := "Your email has been changed"
	textBody := fmt.Sprintf("Your account email has been changed from %s to %s. If you did not perform this action, please contact support.", oldEmail, newEmail)
	htmlBody := fmt.Sprintf(
		`<div>
			<p>Hello %s,</p>
			<p>Your account email has been changed from %s to %s. If you did not perform this action, please contact support immediately.</p>
		</div>`,
		user.Email,
		oldEmail,
		newEmail,
	)

	if err := uc.MailerService.SendEmail(ctx, newEmail, subject, textBody, htmlBody); err != nil {
		uc.Logger.Error("failed to send email to new address", "err", err)
	}

	if err := uc.MailerService.SendEmail(ctx, oldEmail, subject, textBody, htmlBody); err != nil {
		uc.Logger.Error("failed to send email to old address", "err", err)
	}

	return nil
}

func (uc *ChangeEmailUseCase) publishEmailChangedEvent(user *models.User, oldEmail string, newEmail string) {
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
