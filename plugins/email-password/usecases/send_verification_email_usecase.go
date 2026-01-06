package usecases

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type SendVerificationEmailUseCase struct {
	GlobalConfig        *models.Config
	PluginConfig        types.EmailPasswordPluginConfig
	Logger              models.Logger
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
}

func (uc *SendVerificationEmailUseCase) Send(ctx context.Context, email string, callbackURL *string) error {
	reqCtx, _ := models.GetRequestContext(ctx)

	if !uc.PluginConfig.RequireEmailVerification {
		uc.Logger.Debug("email verification is not enabled")
		return nil
	}

	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	user, err := uc.UserService.GetByEmail(ctx, email)
	if err != nil {
		uc.Logger.Error("failed to fetch user", map[string]any{"error": err.Error(), "email": email})
		// Don't expose internal errors to prevent enumeration attacks
		return nil
	}

	if user == nil {
		uc.Logger.Debug("user not found", map[string]any{"email": email})
		// For security, we don't return an error indicating the user doesn't exist
		return nil
	}

	if user.EmailVerified {
		uc.Logger.Debug("email already verified", map[string]any{"user_id": user.ID})
		return nil
	}

	token, err := uc.TokenService.Generate()
	if err != nil {
		uc.Logger.Error("failed to generate verification token", map[string]any{"error": err.Error(), "user_id": user.ID})
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	hashedToken := uc.TokenService.Hash(token)

	if err := uc.VerificationService.DeleteByUserIDAndType(ctx, user.ID, models.TypeEmailVerification); err != nil {
		uc.Logger.Error("failed to delete existing verification tokens", map[string]any{"error": err.Error(), "user_id": user.ID})
		// Continue anyway - the new token will still be created
	}

	if _, err := uc.VerificationService.Create(ctx, user.ID, hashedToken, models.TypeEmailVerification, user.Email, uc.PluginConfig.ExpiresIn); err != nil {
		uc.Logger.Error("failed to create verification token", map[string]any{"error": err.Error(), "user_id": user.ID})
		return fmt.Errorf("failed to create verification token: %w", err)
	}

	verificationLink := util.BuildVerificationURL(
		uc.GlobalConfig.BaseURL,
		uc.GlobalConfig.BasePath,
		token,
		callbackURL,
	)

	if uc.PluginConfig.SendEmailVerification != nil {
		err := uc.PluginConfig.SendEmailVerification(
			types.SendEmailVerificationParams{
				User:  *user,
				URL:   verificationLink,
				Token: token,
			},
			reqCtx,
		)
		if err != nil {
			uc.Logger.Error("custom email verification sender failed", map[string]any{"error": err.Error(), "user_id": user.ID})
			return err
		}
		return nil
	}

	subject := "Verify your email"
	textBody := fmt.Sprintf("Verify your email by clicking the following link: %s.", verificationLink)
	htmlBody := fmt.Sprintf(
		"<p>Hello %s,</p><p>Please verify your email address by clicking the following link:</p><p><a href=\"%s\">%s</a></p><p>If you did not request this, please ignore this email.</p>",
		user.Email,
		verificationLink,
		verificationLink,
	)
	if err := uc.MailerService.SendEmail(ctx, user.Email, subject, textBody, htmlBody); err != nil {
		uc.Logger.Error("failed to send verification email", map[string]any{"error": err.Error(), "user_id": user.ID})
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}
