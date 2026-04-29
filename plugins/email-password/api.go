package email_password

import (
	"context"
	"encoding/json"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

func BuildAPI(plugin *EmailPasswordPlugin) *API {
	useCases := BuildUseCases(plugin)
	return &API{useCases: useCases}
}

func BuildUseCases(p *EmailPasswordPlugin) *usecases.UseCases {
	signUpUseCase := usecases.NewSignUpUseCase(p.globalConfig, p.pluginConfig, p.logger, p.userService, p.accountService, p.sessionService, p.tokenService, p.passwordService, p.ctx.EventBus)
	signInUseCase := usecases.NewSignInUseCase(p.globalConfig, p.pluginConfig, p.logger, p.userService, p.accountService, p.sessionService, p.tokenService, p.passwordService, p.ctx.EventBus)
	verifyEmailUseCase := usecases.NewVerifyEmailUseCase(p.pluginConfig, p.logger, p.userService, p.accountService, p.verificationService, p.tokenService, p.mailerService, p.ctx.EventBus)
	sendEmailVerificationUseCase := usecases.NewSendEmailVerificationUseCase(p.globalConfig, p.pluginConfig, p.logger, p.userService, p.verificationService, p.tokenService, p.mailerService)
	requestPasswordResetUseCase := usecases.NewRequestPasswordResetUseCase(p.logger, p.globalConfig, p.pluginConfig, p.userService, p.verificationService, p.tokenService, p.mailerService)
	changePasswordUseCase := usecases.NewChangePasswordUseCase(p.logger, p.pluginConfig, p.userService, p.accountService, p.verificationService, p.tokenService, p.passwordService, p.mailerService, p.ctx.EventBus)
	requestEmailChangeUseCase := usecases.NewRequestEmailChangeUseCase(p.logger, p.globalConfig, p.pluginConfig, p.userService, p.verificationService, p.tokenService, p.mailerService)

	return &usecases.UseCases{
		SignUpUseCase:                signUpUseCase,
		SignInUseCase:                signInUseCase,
		VerifyEmailUseCase:           verifyEmailUseCase,
		SendEmailVerificationUseCase: sendEmailVerificationUseCase,
		RequestPasswordResetUseCase:  requestPasswordResetUseCase,
		ChangePasswordUseCase:        changePasswordUseCase,
		RequestEmailChangeUseCase:    requestEmailChangeUseCase,
	}
}

type API struct {
	useCases *usecases.UseCases
}

func (a *API) SignUp(
	ctx context.Context,
	name string,
	email string,
	password string,
	image *string,
	metadata json.RawMessage,
	callbackURL *string,
	ipAddress *string,
	userAgent *string,
) (*types.SignUpResult, error) {
	return a.useCases.SignUpUseCase.SignUp(ctx, name, email, password, image, metadata, callbackURL, ipAddress, userAgent)
}

func (a *API) SignIn(
	ctx context.Context,
	email string,
	password string,
	callbackURL *string,
	ipAddress *string,
	userAgent *string,
) (*types.SignInResult, error) {
	return a.useCases.SignInUseCase.SignIn(ctx, email, password, callbackURL, ipAddress, userAgent)
}

func (a *API) VerifyEmail(ctx context.Context, tokenStr string) (models.VerificationType, error) {
	return a.useCases.VerifyEmailUseCase.VerifyEmail(ctx, tokenStr)
}

func (a *API) SendEmailVerification(ctx context.Context, userID string, callbackURL *string) error {
	return a.useCases.SendEmailVerificationUseCase.Send(ctx, userID, callbackURL)
}

func (a *API) RequestPasswordReset(ctx context.Context, email string, callbackURL *string) error {
	return a.useCases.RequestPasswordResetUseCase.RequestReset(ctx, email, callbackURL)
}

func (a *API) ChangePassword(ctx context.Context, tokenStr string, newPassword string) error {
	return a.useCases.ChangePasswordUseCase.ChangePassword(ctx, tokenStr, newPassword)
}

func (a *API) RequestEmailChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error {
	return a.useCases.RequestEmailChangeUseCase.RequestChange(ctx, userID, newEmail, callbackURL)
}
