package usecases

import (
	"context"
	"fmt"
	"time"

	emailconstants "github.com/Authula/authula/internal/email/constants"
	emailtmpl "github.com/Authula/authula/internal/email/template"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/utils"
	rootservices "github.com/Authula/authula/services"
)

type requestEmailChangeUseCase struct {
	Logger               models.Logger
	GlobalConfig         *models.Config
	PluginConfig         types.EmailPasswordPluginConfig
	UserService          rootservices.UserService
	VerificationService  rootservices.VerificationService
	TokenService         rootservices.TokenService
	MailerService        rootservices.MailerService
	EmailTemplateManager *emailtmpl.Manager
}

func NewRequestEmailChangeUseCase(
	logger models.Logger,
	globalConfig *models.Config,
	pluginConfig types.EmailPasswordPluginConfig,
	userService rootservices.UserService,
	verificationService rootservices.VerificationService,
	tokenService rootservices.TokenService,
	mailerService rootservices.MailerService,
	emailTemplateManager *emailtmpl.Manager,
) RequestEmailChangeUseCase {
	return &requestEmailChangeUseCase{Logger: logger, GlobalConfig: globalConfig, PluginConfig: pluginConfig, UserService: userService, VerificationService: verificationService, TokenService: tokenService, MailerService: mailerService, EmailTemplateManager: emailTemplateManager}
}

func (uc *requestEmailChangeUseCase) RequestChange(
	ctx context.Context,
	userID string,
	newEmail string,
	callbackURL *string,
) error {
	reqCtx, _ := models.GetRequestContext(ctx)

	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return constants.ErrUserNotFound
	}

	existing, err := uc.UserService.GetByEmail(ctx, newEmail)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != user.ID {
		return constants.ErrEmailAlreadyExists
	}

	token, err := uc.TokenService.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate change token: %w", err)
	}

	hashedToken := uc.TokenService.Hash(token)

	if _, err = uc.VerificationService.Create(
		ctx,
		user.ID,
		hashedToken,
		models.TypeEmailResetRequest,
		newEmail,
		uc.PluginConfig.RequestEmailChangeExpiresIn,
	); err != nil {
		return err
	}

	verificationLink := utils.BuildVerificationURL(
		uc.GlobalConfig.BaseURL,
		uc.GlobalConfig.BasePath,
		token,
		callbackURL,
	)
	callbackHandled := false

	if uc.PluginConfig.SendRequestEmailChangeEmail != nil {
		err := uc.PluginConfig.SendRequestEmailChangeEmail(
			types.SendRequestEmailChangeEmailParams{
				User:     *user,
				URL:      verificationLink,
				Token:    token,
				NewEmail: newEmail,
				OldEmail: user.Email,
			},
			reqCtx,
		)

		if err != nil {
			uc.Logger.Error("failed to send request email change via plugin callback", "err", err.Error())
		} else {
			callbackHandled = true
		}
	}

	if !callbackHandled && uc.MailerService != nil {
		go func() {
			detachedCtx := context.WithoutCancel(ctx)
			taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
			defer cancel()

			if err := uc.sendRequestEmailChangeEmail(taskCtx, user, newEmail, verificationLink); err != nil {
				uc.Logger.Error("failed to send request email change via built-in email service", "err", err.Error())
			}
		}()
	}

	return nil
}

func (uc *requestEmailChangeUseCase) sendRequestEmailChangeEmail(ctx context.Context, user *models.User, newEmail string, verificationLink string) error {
	subject, textBody, htmlBody, err := uc.EmailTemplateManager.Render(emailconstants.EmailChangeRequestEmailTemplateName, types.EmailChangeRequestContext{
		CommonContext: emailtmpl.NewCommonContext(uc.GlobalConfig.AppName, uc.GlobalConfig.BaseURL),
		UserEmail:     user.Email,
		OldEmail:      user.Email,
		NewEmail:      newEmail,
		ChangeLink:    verificationLink,
		Expiry:        uc.PluginConfig.RequestEmailChangeExpiresIn,
	})
	if err != nil {
		uc.Logger.Error("failed to render email change request template", "err", err.Error())
		return err
	}
	return uc.MailerService.SendEmail(ctx, newEmail, subject, textBody, htmlBody)
}
