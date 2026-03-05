package usecases

import (
	"context"

	twofactor "github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type getTOTPURIUseCase struct {
	AccountService  rootservices.AccountService
	PasswordService rootservices.PasswordService
	TokenService    rootservices.TokenService
	TOTPService     *services.TOTPService
	Repo            *twofactor.TwoFactorRepository
	Config          *types.TwoFactorPluginConfig
}

func NewGetTOTPURIUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	tokenService rootservices.TokenService,
	totpService *services.TOTPService,
	repo *twofactor.TwoFactorRepository,
	config *types.TwoFactorPluginConfig,
) GetTOTPURIUseCase {
	return &getTOTPURIUseCase{
		AccountService:  accountService,
		PasswordService: passwordService,
		TokenService:    tokenService,
		TOTPService:     totpService,
		Repo:            repo,
		Config:          config,
	}
}

func (uc *getTOTPURIUseCase) GetTOTPURI(ctx context.Context, userID, password, email string) (string, error) {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return "", err
	}

	// Get two-factor record
	record, err := uc.Repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", constants.ErrTwoFactorNotEnabled
	}

	// Decrypt secret
	secret, err := uc.TokenService.Decrypt(record.Secret)
	if err != nil {
		return "", err
	}

	// Build URI
	issuer := uc.Config.Issuer
	return uc.TOTPService.BuildURI(secret, issuer, email), nil
}
