package auth

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type AuthApiImpl struct {
	useCases    UseCases
	authService *Service
}

func NewApi(useCases UseCases, authService *Service) models.AuthApi {
	return &AuthApiImpl{
		useCases:    useCases,
		authService: authService,
	}
}

func (a *AuthApiImpl) Services() *models.AuthServices {
	return &models.AuthServices{
		Users:         a.authService.UserService,
		Accounts:      a.authService.AccountService,
		Sessions:      a.authService.SessionService,
		Verifications: a.authService.VerificationService,
		Passwords:     a.authService.PasswordService,
		Tokens:        a.authService.TokenService,
		RateLimits:    a.authService.RateLimitService,
		Mailers:       a.authService.MailerService,
	}
}

func (a *AuthApiImpl) SignUpWithEmailAndPassword(ctx context.Context, name string, email string, password string, callbackURL *string) (*models.SignUpResult, error) {
	return a.useCases.SignUpUseCase.SignUpWithEmailAndPassword(ctx, name, email, password, callbackURL)
}

func (a *AuthApiImpl) SignInWithEmailAndPassword(ctx context.Context, email string, password string, callbackURL *string) (*models.SignInResult, error) {
	return a.useCases.SignInUseCase.SignInWithEmailAndPassword(ctx, email, password, callbackURL)
}

func (a *AuthApiImpl) SignOut(ctx context.Context, sessionToken string) error {
	return a.useCases.SignOutUseCase.SignOut(ctx, sessionToken)
}

func (a *AuthApiImpl) VerifyEmail(ctx context.Context, rawToken string) (*models.VerifyEmailResult, error) {
	return a.useCases.VerifyEmailUseCase.VerifyEmail(ctx, rawToken)
}

func (a *AuthApiImpl) SendEmailVerification(ctx context.Context, userID string, callbackURL *string) error {
	return a.useCases.SendEmailVerificationUseCase.SendEmailVerification(ctx, userID, callbackURL)
}

func (a *AuthApiImpl) ResetPassword(ctx context.Context, email string, callbackURL *string) error {
	return a.useCases.ResetPasswordUseCase.ResetPassword(ctx, email, callbackURL)
}

func (a *AuthApiImpl) ChangePassword(ctx context.Context, rawToken string, newPassword string) error {
	return a.useCases.ChangePasswordUseCase.ChangePassword(ctx, rawToken, newPassword)
}

func (a *AuthApiImpl) EmailChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error {
	return a.useCases.EmailChangeUseCase.EmailChange(ctx, userID, newEmail, callbackURL)
}

func (a *AuthApiImpl) GetMe(ctx context.Context, userID string) (*models.MeResult, error) {
	return a.useCases.MeUseCase.GetMe(ctx, userID)
}

func (a *AuthApiImpl) PrepareOAuth2Login(ctx context.Context, providerName string) (*models.OAuth2LoginResult, error) {
	return a.useCases.OAuth2UseCase.PrepareOAuth2Login(ctx, providerName)
}

func (a *AuthApiImpl) SignInWithOAuth2(ctx context.Context, providerName string, code string, state string, verifier *string) (*models.SignInResult, error) {
	return a.useCases.OAuth2UseCase.SignInWithOAuth2(ctx, providerName, code, state, verifier)
}
