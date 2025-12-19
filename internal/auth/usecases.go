package auth

import (
	changepassword "github.com/GoBetterAuth/go-better-auth/internal/auth/change-password"
	emailchange "github.com/GoBetterAuth/go-better-auth/internal/auth/email-change"
	me "github.com/GoBetterAuth/go-better-auth/internal/auth/me"
	oauth2 "github.com/GoBetterAuth/go-better-auth/internal/auth/oauth2"
	resetpassword "github.com/GoBetterAuth/go-better-auth/internal/auth/reset-password"
	sendemailverification "github.com/GoBetterAuth/go-better-auth/internal/auth/send-email-verification"
	signin "github.com/GoBetterAuth/go-better-auth/internal/auth/sign-in"
	signout "github.com/GoBetterAuth/go-better-auth/internal/auth/sign-out"
	signup "github.com/GoBetterAuth/go-better-auth/internal/auth/sign-up"
	verifyemail "github.com/GoBetterAuth/go-better-auth/internal/auth/verify-email"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type UseCases struct {
	SignUpUseCase                signup.SignUpUseCase
	SignInUseCase                signin.SignInUseCase
	SignOutUseCase               signout.SignOutUseCase
	VerifyEmailUseCase           verifyemail.VerifyEmailUseCase
	SendEmailVerificationUseCase sendemailverification.SendEmailVerificationUseCase
	ResetPasswordUseCase         resetpassword.ResetPasswordUseCase
	ChangePasswordUseCase        changepassword.ChangePasswordUseCase
	EmailChangeUseCase           emailchange.EmailChangeUseCase
	MeUseCase                    me.MeUseCase
	OAuth2UseCase                oauth2.OAuth2UseCase
}

func NewUseCases(config *models.Config, authService *Service) *UseCases {
	signInUseCase := signin.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.AccountService,
		authService.SessionService,
		authService.TokenService,
		authService.VerificationService,
		authService.MailerService,
		authService.PasswordService,
		authService.EventEmitter,
	)

	signUpUseCase := signup.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.AccountService,
		authService.SessionService,
		authService.TokenService,
		authService.VerificationService,
		authService.PasswordService,
		authService.EventEmitter,
	)

	signOutUseCase := signout.New(
		config,
		config.Logger.Logger,
		authService.SessionService,
		authService.TokenService,
	)

	verifyEmailUseCase := verifyemail.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.TokenService,
		authService.VerificationService,
		authService.EventEmitter,
	)

	sendEmailVerificationUseCase := sendemailverification.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.TokenService,
		authService.VerificationService,
		authService.MailerService,
	)

	resetPasswordUseCase := resetpassword.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.VerificationService,
		authService.TokenService,
		authService.MailerService,
	)

	changePasswordUseCase := changepassword.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.AccountService,
		authService.VerificationService,
		authService.TokenService,
		authService.PasswordService,
		authService.EventEmitter,
	)

	emailChangeUseCase := emailchange.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.VerificationService,
		authService.TokenService,
		authService.MailerService,
	)

	meUseCase := me.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.SessionService,
	)

	oauth2UseCase := oauth2.New(
		config,
		config.Logger.Logger,
		authService.UserService,
		authService.AccountService,
		authService.SessionService,
		authService.TokenService,
		authService.OAuth2ProviderRegistry,
	)

	return &UseCases{
		SignUpUseCase:                signUpUseCase,
		SignInUseCase:                signInUseCase,
		SignOutUseCase:               signOutUseCase,
		VerifyEmailUseCase:           verifyEmailUseCase,
		SendEmailVerificationUseCase: sendEmailVerificationUseCase,
		ResetPasswordUseCase:         resetPasswordUseCase,
		ChangePasswordUseCase:        changePasswordUseCase,
		EmailChangeUseCase:           emailChangeUseCase,
		MeUseCase:                    meUseCase,
		OAuth2UseCase:                oauth2UseCase,
	}
}
