package usecases

import (
	"context"
	"fmt"
	"time"

	emailconstants "github.com/Authula/authula/internal/email/constants"
	emailtmpl "github.com/Authula/authula/internal/email/template"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/utils"
	rootservices "github.com/Authula/authula/services"
)

type requestPasswordResetUseCase struct {
	Logger               models.Logger
	GlobalConfig         *models.Config
	PluginConfig         types.EmailPasswordPluginConfig
	UserService          rootservices.UserService
	VerificationService  rootservices.VerificationService
	TokenService         rootservices.TokenService
	MailerService        rootservices.MailerService
	EmailTemplateManager *emailtmpl.Manager
}

func NewRequestPasswordResetUseCase(
	logger models.Logger,
	globalConfig *models.Config,
	pluginConfig types.EmailPasswordPluginConfig,
	userService rootservices.UserService,
	verificationService rootservices.VerificationService,
	tokenService rootservices.TokenService,
	mailerService rootservices.MailerService,
	emailTemplateManager *emailtmpl.Manager,
) RequestPasswordResetUseCase {
	return &requestPasswordResetUseCase{
		Logger:               logger,
		GlobalConfig:         globalConfig,
		PluginConfig:         pluginConfig,
		UserService:          userService,
		VerificationService:  verificationService,
		TokenService:         tokenService,
		MailerService:        mailerService,
		EmailTemplateManager: emailTemplateManager,
	}
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

			if err := uc.sendRequestPasswordResetEmail(taskCtx, user, verificationLink); err != nil {
				uc.Logger.Error("failed to send password reset email via built-in email service", "err", err.Error())
			}
		}()
	}

	return nil
}

func (uc *requestPasswordResetUseCase) sendRequestPasswordResetEmail(ctx context.Context, user *models.User, verificationLink string) error {
	subject, textBody, htmlBody, err := uc.EmailTemplateManager.Render(emailconstants.PasswordResetRequestEmailTemplateName, types.PasswordResetContext{
		CommonContext: emailtmpl.NewCommonContext(uc.GlobalConfig.AppName, uc.GlobalConfig.BaseURL),
		UserEmail:     user.Email,
		ResetLink:     verificationLink,
		Expiry:        uc.PluginConfig.PasswordResetExpiresIn,
	})
	if err != nil {
		uc.Logger.Error("failed to render password reset template", "err", err.Error())
		return err
	}
	return uc.MailerService.SendEmail(ctx, user.Email, subject, textBody, htmlBody)
}
