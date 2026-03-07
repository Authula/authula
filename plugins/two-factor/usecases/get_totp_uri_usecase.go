package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type getTOTPURIUseCase struct {
	UserService     rootservices.UserService
	AccountService  rootservices.AccountService
	PasswordService rootservices.PasswordService
	TokenService    rootservices.TokenService
	TOTPService     *services.TOTPService
	TwoFactorRepo   *repository.TwoFactorRepository
	Config          *types.TwoFactorPluginConfig
}

func NewGetTOTPURIUseCase(
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	tokenService rootservices.TokenService,
	totpService *services.TOTPService,
	twoFactorRepo *repository.TwoFactorRepository,
	config *types.TwoFactorPluginConfig,
) GetTOTPURIUseCase {
	return &getTOTPURIUseCase{
		UserService:     userService,
		AccountService:  accountService,
		PasswordService: passwordService,
		TokenService:    tokenService,
		TOTPService:     totpService,
		TwoFactorRepo:   twoFactorRepo,
		Config:          config,
	}
}

func (uc *getTOTPURIUseCase) GetTOTPURI(ctx context.Context, userID, password string) (string, error) {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return "", err
	}

	// Get two-factor record
	record, err := uc.TwoFactorRepo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", constants.ErrTwoFactorNotEnabled
	}

	// Fetch user to get email
	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", constants.ErrUserNotFound
	}

	// Decrypt secret
	secret, err := uc.TokenService.Decrypt(record.Secret)
	if err != nil {
		return "", err
	}

	// Build URI
	issuer := uc.Config.Issuer
	return uc.TOTPService.BuildURI(secret, issuer, user.Email), nil
}
