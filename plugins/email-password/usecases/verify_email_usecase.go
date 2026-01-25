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

type VerifyEmailUseCase struct {
	PluginConfig        types.EmailPasswordPluginConfig
	Logger              models.Logger
	UserService         rootservices.UserService
	AccountService      rootservices.AccountService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
	EventBus            models.EventBus
}

func (uc *VerifyEmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) (models.VerificationType, error) {
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
	case models.TypePasswordResetRequest:
	case models.TypeEmailResetRequest:
		if err := uc.handleEmailChangeVerificationEmail(ctx, *verification.UserID, tokenStr, verification.Identifier); err != nil {
			return "", err
		}
	default:
		return "", constants.ErrInvalidEmailVerificationType
	}

	return verification.Type, nil
}

func (uc *VerifyEmailUseCase) handleEmailVerification(ctx context.Context, user *models.User, tokenID string) error {
	user.EmailVerified = true
	if _, err := uc.UserService.Update(ctx, user); err != nil {
		return err
	}

	if err := uc.VerificationService.Delete(ctx, tokenID); err != nil {
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

func (uc *VerifyEmailUseCase) handleEmailChangeVerificationEmail(
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

	go func() {
		detachedCtx := context.WithoutCancel(ctx)
		taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
		defer cancel()

		if err := uc.sendChangedEmailNotifications(taskCtx, oldEmail, newEmail); err != nil {
			uc.Logger.Error("failed to send changed email notifications", "err", err)
		}
	}()

	uc.publishEmailChangedEvent(user, oldEmail, newEmail)

	return nil
}

func (uc *VerifyEmailUseCase) sendChangedEmailNotifications(ctx context.Context, oldEmail string, newEmail string) error {
	subject := "Your email has been changed"
	textBody := fmt.Sprintf("Your account email has been changed from %s to %s. If you did not perform this action, please contact support.", oldEmail, newEmail)

	if err := uc.MailerService.SendEmail(ctx, oldEmail, subject, textBody, getHtmlBody(oldEmail, oldEmail, newEmail)); err != nil {
		uc.Logger.Error("failed to send email to old address", "err", err)
	}

	if err := uc.MailerService.SendEmail(ctx, newEmail, subject, textBody, getHtmlBody(newEmail, oldEmail, newEmail)); err != nil {
		uc.Logger.Error("failed to send email to new address", "err", err)
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

func (uc *VerifyEmailUseCase) publishEmailChangedEvent(user *models.User, oldEmail string, newEmail string) {
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
