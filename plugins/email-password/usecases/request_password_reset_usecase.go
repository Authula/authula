package usecases

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type RequestPasswordResetUseCase struct {
	GlobalConfig        *models.Config
	PluginConfig        types.EmailPasswordPluginConfig
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
	MailerService       rootservices.MailerService
}

func (uc *RequestPasswordResetUseCase) RequestReset(
	ctx context.Context,
	email string,
	callbackURL *string,
) error {
	user, err := uc.UserService.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return nil
	}

	token, err := uc.TokenService.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	hashedToken := uc.TokenService.Hash(token)

	if _, err = uc.VerificationService.Create(ctx, user.ID, hashedToken, models.TypePasswordResetRequest, user.Email, uc.PluginConfig.ExpiresIn); err != nil {
		// swallow error to avoid enumeration
		return nil
	}

	resetURL := util.BuildVerificationURL(
		uc.GlobalConfig.BaseURL,
		uc.GlobalConfig.BasePath,
		token,
		callbackURL,
	)

	subject := "Reset Your Password"
	textBody := fmt.Sprintf("Please reset your password by clicking the following link: %s.", resetURL)
	htmlBody := `<html>
<body>
	<h2>Password Reset Request</h2>
	<p>Hello, ` + user.Email + `</p>
	<p>We received a request to reset your password. If you made this request, please click the link below to reset your password:</p>
	<p><a href="` + resetURL + `">Reset Password</a></p>
	<p>If you did not request a password reset, please ignore this email. Your password will remain unchanged.</p>
	<p>This link will expire in ` + fmt.Sprintf("%d", int(uc.PluginConfig.ExpiresIn.Hours())) + ` hours.</p>
	<p>If you're having trouble clicking the link, copy and paste this URL into your browser: ` + resetURL + `</p>
	<p>&copy; 2023 GoBetterAuth. All rights reserved.</p>
</body>
</html>`
	go uc.MailerService.SendEmail(
		context.Background(),
		user.Email,
		subject,
		textBody,
		htmlBody,
	)

	return nil
}
