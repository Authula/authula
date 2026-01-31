package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type RequestEmailChangeUseCase struct {
	Logger              models.Logger
	GlobalConfig        *models.Config
	PluginConfig        types.EmailPasswordPluginConfig
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
}

func (uc *RequestEmailChangeUseCase) RequestChange(
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

	verificationLink := util.BuildVerificationURL(
		uc.GlobalConfig.BaseURL,
		uc.GlobalConfig.BasePath,
		token,
		callbackURL,
	)

	if uc.PluginConfig.SendRequestEmailChangeEmail != nil {
		if err := uc.PluginConfig.SendRequestEmailChangeEmail(
			types.SendRequestEmailChangeEmailParams{
				User:     *user,
				URL:      verificationLink,
				Token:    token,
				NewEmail: newEmail,
				OldEmail: user.Email,
			},
			reqCtx,
		); err != nil {
			uc.Logger.Error("failed to send request email change via hook", "err", err)
			return err
		}
		return nil
	}

	go func() {
		detachedCtx := context.WithoutCancel(ctx)
		taskCtx, cancel := context.WithTimeout(detachedCtx, 15*time.Second)
		defer cancel()

		if err := uc.sendRequestEmailChangeEmail(taskCtx, user, newEmail, verificationLink); err != nil {
			uc.Logger.Error(err.Error())
		}
	}()

	return nil
}

func (uc *RequestEmailChangeUseCase) sendRequestEmailChangeEmail(ctx context.Context, user *models.User, newEmail string, verificationLink string) error {
	expiryInHours := int(uc.PluginConfig.RequestEmailChangeExpiresIn.Hours())
	hoursText := "hours"
	if expiryInHours < 2 {
		hoursText = "hour"
	}
	subject := "Confirm Your Email Change"
	textBody := fmt.Sprintf("Please confirm your email change to %s by clicking the following link: %s.", newEmail, verificationLink)
	htmlBody := fmt.Sprintf(
		`<html>
			<body>
				<p>Hello, %s</p>
				<p>We received a request to change your account email to %s. If you made this request, please click the link below to confirm the change:</p>
				<p><a href="%s">Confirm Email Change</a></p>
				<p>This link will expire in %d %s.</p>
				<p>If you did not request an email change, please ignore this email.</p>
		</body>
	</html>`,
		user.Email,
		newEmail,
		verificationLink,
		expiryInHours,
		hoursText,
	)
	return uc.MailerService.SendEmail(ctx, newEmail, subject, textBody, htmlBody)
}
