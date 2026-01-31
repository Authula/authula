package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type SendEmailVerificationUseCase struct {
	GlobalConfig        *models.Config
	PluginConfig        types.EmailPasswordPluginConfig
	Logger              models.Logger
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
}

func (uc *SendEmailVerificationUseCase) Send(ctx context.Context, email string, callbackURL *string) error {
	reqCtx, _ := models.GetRequestContext(ctx)

	if !uc.PluginConfig.RequireEmailVerification {
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
		// For security, we don't return an error indicating the user doesn't exist
		return nil
	}

	if user.EmailVerified {
		return nil
	}

	token, err := uc.TokenService.Generate()
	if err != nil {
		uc.Logger.Error(err.Error())
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	hashedToken := uc.TokenService.Hash(token)

	if err := uc.VerificationService.DeleteByUserIDAndType(ctx, user.ID, models.TypeEmailVerification); err != nil {
		uc.Logger.Error(err.Error())
		// Continue anyway - the new token will still be created
	}

	if _, err := uc.VerificationService.Create(
		ctx,
		user.ID,
		hashedToken,
		models.TypeEmailVerification,
		user.Email,
		uc.PluginConfig.EmailVerificationExpiresIn,
	); err != nil {
		uc.Logger.Error(err.Error())
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
			uc.Logger.Error(err.Error())
			return err
		}
		return nil
	}

	go func() {
		detachedCtx := context.WithoutCancel(ctx)
		taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
		defer cancel()

		if err := uc.sendEmailVerification(taskCtx, user, verificationLink); err != nil {
			uc.Logger.Error(err.Error())
		}
	}()

	return nil
}

func (uc *SendEmailVerificationUseCase) sendEmailVerification(ctx context.Context, user *models.User, verificationLink string) error {
	expiryInHours := int(uc.PluginConfig.EmailVerificationExpiresIn.Hours())
	hoursText := "hours"
	if expiryInHours < 2 {
		hoursText = "hour"
	}
	subject := "Verify your email"
	textBody := fmt.Sprintf("Verify your email by clicking the following link: %s.", verificationLink)
	htmlBody := fmt.Sprintf(
		strings.TrimSpace(
			`<div>
				<p>Hello %s,</p>
				<p>Please verify your email address by clicking the following link: <a href="%s">Verify your email</a></p>
				<p>This link will expire in %d %s.</p>
				<p>If you did not request this, please ignore this email.</p>
			</div>`,
		),
		user.Email,
		verificationLink,
		expiryInHours,
		hoursText,
	)
	return uc.MailerService.SendEmail(ctx, user.Email, subject, textBody, htmlBody)
}
