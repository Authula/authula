package usecases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
	rootservices "github.com/Authula/authula/services"
)

type SignInUseCaseImpl struct {
	GlobalConfig        *models.Config
	PluginConfig        *types.MagicLinkPluginConfig
	Logger              models.Logger
	UserService         rootservices.UserService
	AccountService      rootservices.AccountService
	TokenService        rootservices.TokenService
	VerificationService rootservices.VerificationService
	MailerService       rootservices.MailerService
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

			if err := uc.sendMagicLinkVerificationEmail(taskCtx, existingUser, verificationURL); err != nil {
				uc.Logger.Error("failed to send magic link verification email via built-in email service", "err", err.Error())
			}
		}()
	}

	return &types.SignInResult{
		Token: token,
	}, nil
}

func (uc *SignInUseCaseImpl) sendMagicLinkVerificationEmail(ctx context.Context, user *models.User, verificationURL string) error {
	expiresIn := util.FormatDuration(uc.PluginConfig.ExpiresIn)
	greeting := user.Email
	if user.Name != "" {
		greeting = user.Name
	}
	subject := fmt.Sprintf("Sign in to %s with your magic link", uc.GlobalConfig.AppName)
	textBody := fmt.Sprintf(
		"Hi %s,\n\nClick the link below to sign in to your account:\n\n%s\n\nThis link will expire in %s.\n\nIf you didn't request this, please ignore this email.\n\n",
		greeting,
		verificationURL,
		expiresIn,
	)
	htmlBody := fmt.Sprintf(
		`<p>Hi %s,</p>
		<p>Click the link below to sign in to your account:</p>
		<p><a href="%s">Sign in</a></p>
		<p>This link will expire in %s.</p>
		<p>If you didn't request this, please ignore this email.</p>`,
		greeting,
		verificationURL,
		expiresIn,
	)
	return uc.MailerService.SendEmail(ctx, user.Email, subject, textBody, htmlBody)
}
