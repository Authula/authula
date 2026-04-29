package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/utils"
	rootservices "github.com/Authula/authula/services"
)

type requestPasswordResetUseCase struct {
	Logger              models.Logger
	GlobalConfig        *models.Config
	PluginConfig        types.EmailPasswordPluginConfig
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
}

func NewRequestPasswordResetUseCase(
	logger models.Logger,
	globalConfig *models.Config,
	pluginConfig types.EmailPasswordPluginConfig,
	userService rootservices.UserService,
	verificationService rootservices.VerificationService,
	tokenService rootservices.TokenService,
	mailerService rootservices.MailerService,
) RequestPasswordResetUseCase {
	return &requestPasswordResetUseCase{Logger: logger, GlobalConfig: globalConfig, PluginConfig: pluginConfig, UserService: userService, VerificationService: verificationService, TokenService: tokenService, MailerService: mailerService}
}

func (uc *requestPasswordResetUseCase) RequestReset(
	ctx context.Context,
	email string,
	callbackURL *string,
) error {
	reqCtx, _ := models.GetRequestContext(ctx)

	user, err := uc.UserService.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return nil
	}

	token, err := uc.TokenService.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	hashedToken := uc.TokenService.Hash(token)

	if _, err = uc.VerificationService.Create(
		ctx,
		user.ID,
		hashedToken,
		models.TypePasswordResetRequest,
		user.Email,
		uc.PluginConfig.PasswordResetExpiresIn,
	); err != nil {
		// swallow error to avoid enumeration
		return nil
	}

	verificationLink := utils.BuildVerificationURL(
		uc.GlobalConfig.BaseURL,
		uc.GlobalConfig.BasePath,
		token,
		callbackURL,
	)
	callbackHandled := false

	if uc.PluginConfig.SendPasswordResetEmail != nil {
		err := uc.PluginConfig.SendPasswordResetEmail(
			types.SendPasswordResetEmailParams{
				User:  *user,
				URL:   verificationLink,
				Token: token,
			},
			reqCtx,
		)

		if err != nil {
			uc.Logger.Error("failed to send password reset email via plugin callback", "err", err.Error())
		} else {
			callbackHandled = true
		}
	}

	if !callbackHandled && uc.MailerService != nil {
		go func() {
			detachedCtx := context.WithoutCancel(ctx)
			taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
			defer cancel()

			if err := sendRequestPasswordResetEmail(taskCtx, user, verificationLink, uc.PluginConfig.PasswordResetExpiresIn, uc.MailerService); err != nil {
				uc.Logger.Error("failed to send password reset email via built-in email service", "err", err.Error())
			}
		}()
	}

	return nil
}

func sendRequestPasswordResetEmail(ctx context.Context, user *models.User, verificationLink string, expiresIn time.Duration, mailerService rootservices.MailerService) error {
	hours := int(expiresIn.Hours())
	hoursText := "hours"
	if hours == 1 {
		hoursText = "hour"
	}
	subject := "Reset Your Password"
	textBody := fmt.Sprintf("Please reset your password by clicking the following link: %s.", verificationLink)
	htmlBody := fmt.Sprintf(
		`<html>
			<body>
				<p>Hello, %s</p>
				<p>We received a request to reset your password. If you made this request, please click the link below to reset your password:</p>
				<p><a href="%s">Reset Password</a></p>
				<p>This link will expire in %d %s.</p>
				<p>If you did not request a password reset, please ignore this email. Your password will remain unchanged.</p>
		</body>
	</html>`,
		user.Email,
		verificationLink,
		hours,
		hoursText,
	)
	return mailerService.SendEmail(ctx, user.Email, subject, textBody, htmlBody)
}
