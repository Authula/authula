package usecases

import (
	"context"
	"time"

	emailconstants "github.com/Authula/authula/core/email/constants"
	emailtmpl "github.com/Authula/authula/core/email/template"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type changePasswordUseCase struct {
	GlobalConfig         *models.Config
	Logger               models.Logger
	PluginConfig         types.EmailPasswordPluginConfig
	UserService          rootservices.UserService
	AccountService       rootservices.AccountService
	VerificationService  rootservices.VerificationService
	TokenService         rootservices.TokenService
	PasswordService      rootservices.PasswordService
	MailerService        rootservices.MailerService
	EventBus             models.EventBus
	EmailTemplateManager *emailtmpl.Manager
}

func NewChangePasswordUseCase(
	globalConfig *models.Config,
	logger models.Logger,
	pluginConfig types.EmailPasswordPluginConfig,
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	verificationService rootservices.VerificationService,
	tokenService rootservices.TokenService,
	passwordService rootservices.PasswordService,
	mailerService rootservices.MailerService,
	eventBus models.EventBus,
	emailTemplateManager *emailtmpl.Manager,
) ChangePasswordUseCase {
	return &changePasswordUseCase{
		GlobalConfig:         globalConfig,
		Logger:               logger,
		PluginConfig:         pluginConfig,
		UserService:          userService,
		AccountService:       accountService,
		VerificationService:  verificationService,
		TokenService:         tokenService,
		PasswordService:      passwordService,
		MailerService:        mailerService,
		EventBus:             eventBus,
		EmailTemplateManager: emailTemplateManager,
	}
}

func (uc *changePasswordUseCase) ChangePassword(ctx context.Context, tokenValue string, newPassword string) error {
	reqCtx, _ := models.GetRequestContext(ctx)

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
		verification.ExpiresAt.Before(time.Now()) ||
		verification.Type != models.TypePasswordResetRequest {
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

	uc.publishChangedPasswordEvent(user)

	callbackHandled := false

	if uc.PluginConfig.SendChangedPasswordEmail != nil {
		err := uc.PluginConfig.SendChangedPasswordEmail(types.SendChangedPasswordEmailParams{
			User: *user,
		}, reqCtx)

		if err != nil {
			uc.Logger.Error("failed to send changed password email via plugin callback", "err", err.Error())
		} else {
			callbackHandled = true
		}
	}

	if !callbackHandled && uc.MailerService != nil {
		go func() {
			detachedCtx := context.WithoutCancel(ctx)
			taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
			defer cancel()

			if err := uc.sendChangedPasswordEmail(taskCtx, user); err != nil {
				uc.Logger.Error("failed to send changed password email via built-in email service", "err", err.Error())
			}
		}()
	}

	return nil
}

func (uc *changePasswordUseCase) sendChangedPasswordEmail(ctx context.Context, user *models.User) error {
	subject, textBody, htmlBody, err := uc.EmailTemplateManager.Render(emailconstants.PasswordChangedEmailTemplateName, types.PasswordChangedContext{
		CommonContext: emailtmpl.NewCommonContext(uc.GlobalConfig.AppName, uc.GlobalConfig.BaseURL),
		UserEmail:     user.Email,
	})
	if err != nil {
		uc.Logger.Error("failed to render password changed template", "err", err.Error())
		return err
	}
	return uc.MailerService.SendEmail(ctx, user.Email, subject, textBody, htmlBody)
}

func (uc *changePasswordUseCase) publishChangedPasswordEvent(user *models.User) {
	util.PublishEventAsync(
		uc.EventBus,
		uc.Logger,
		models.Event{
			ID:        util.GenerateUUID(),
			Type:      constants.EventUserChangedPassword,
			Payload:   util.ToMap(user),
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}
