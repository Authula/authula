package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	emailconstants "github.com/Authula/authula/internal/email/constants"
	emailtmpl "github.com/Authula/authula/internal/email/template"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
	rootservices "github.com/Authula/authula/services"
)

type SignInUseCaseImpl struct {
	GlobalConfig         *models.Config
	PluginConfig         *types.MagicLinkPluginConfig
	Logger               models.Logger
	UserService          rootservices.UserService
	AccountService       rootservices.AccountService
	TokenService         rootservices.TokenService
	VerificationService  rootservices.VerificationService
	MailerService        rootservices.MailerService
	EmailTemplateManager *emailtmpl.Manager
}

func (uc *SignInUseCaseImpl) SignIn(
	ctx context.Context,
	name *string,
	email string,
	callbackURL *string,
) (*types.SignInResult, error) {
	reqCtx, _ := models.GetRequestContext(ctx)

	email = strings.ToLower(email)

	existingUser, err := uc.UserService.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if existingUser == nil {
		if uc.PluginConfig.DisableSignUp {
			return nil, fmt.Errorf("magic link sign up is disabled")
		}

		emptyName := ""
		if name == nil {
			name = &emptyName
		}

		newUser, err := uc.UserService.Create(ctx, *name, email, false, nil, nil)
		if err != nil {
			return nil, err
		}
		existingUser = newUser

		_, err = uc.AccountService.Create(ctx, existingUser.ID, email, models.AuthProviderMagicLink.String(), nil)
		if err != nil {
			return nil, err
		}
	}

	token, err := uc.TokenService.Generate()
	if err != nil {
		return nil, err
	}

	hashedToken := uc.TokenService.Hash(token)
	_, err = uc.VerificationService.Create(
		ctx,
		existingUser.ID,
		hashedToken,
		models.TypeMagicLinkSignInRequest,
		email,
		uc.PluginConfig.ExpiresIn,
	)
	if err != nil {
		return nil, err
	}

	verificationURL := util.BuildActionURL(
		uc.GlobalConfig.BaseURL,
		uc.GlobalConfig.BasePath,
		"/magic-link/verify",
		token,
		callbackURL,
	)

	callbackHandled := false

	if uc.PluginConfig.SendMagicLinkVerificationEmail != nil {
		err := uc.PluginConfig.SendMagicLinkVerificationEmail(types.SendMagicLinkVerificationEmailParams{
			Email: email,
			URL:   verificationURL,
			Token: token,
		}, reqCtx)

		if err != nil {
			uc.Logger.Error("failed to send magic link verification email via plugin callback", "err", err.Error())
		} else {
			callbackHandled = true
		}
	}

	if !callbackHandled && uc.MailerService != nil {
		go func() {
			detachedCtx := context.WithoutCancel(ctx)
			taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
			defer cancel()

			if err := uc.sendMagicLinkVerificationEmail(taskCtx, existingUser.Email, verificationURL); err != nil {
				uc.Logger.Error("failed to send magic link verification email via built-in email service", "err", err.Error())
			}
		}()
	}

	return &types.SignInResult{
		Token: token,
	}, nil
}

func (uc *SignInUseCaseImpl) sendMagicLinkVerificationEmail(ctx context.Context, userEmail string, verificationURL string) error {
	subject, textBody, htmlBody, err := uc.EmailTemplateManager.Render(emailconstants.MagicLinkSignInEmailTemplateName, types.MagicLinkSignInContext{
		CommonContext: emailtmpl.NewCommonContext(uc.GlobalConfig.AppName, uc.GlobalConfig.BaseURL),
		UserEmail:     userEmail,
		MagicLink:     verificationURL,
		Expiry:        uc.PluginConfig.ExpiresIn,
	})
	if err != nil {
		uc.Logger.Error("failed to render magic link sign in template", "err", err.Error())
		return err
	}
	return uc.MailerService.SendEmail(ctx, userEmail, subject, textBody, htmlBody)
}
