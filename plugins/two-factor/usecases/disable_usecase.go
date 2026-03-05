package usecases

import (
	"context"

	twofactor "github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type disableUseCase struct {
	AccountService  rootservices.AccountService
	PasswordService rootservices.PasswordService
	Repo            *twofactor.TwoFactorRepository
}

func NewDisableUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	repo *twofactor.TwoFactorRepository,
) DisableUseCase {
	return &disableUseCase{
		AccountService:  accountService,
		PasswordService: passwordService,
		Repo:            repo,
	}
}

func (uc *disableUseCase) Disable(ctx context.Context, userID, password string) error {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return err
	}

	// Check that 2FA is enabled
	existing, err := uc.Repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return constants.ErrTwoFactorNotEnabled
	}

	// Delete two_factor record
	if err := uc.Repo.DeleteByUserID(ctx, userID); err != nil {
		return err
	}

	// Set user.two_factor_enabled = false
	if err := uc.Repo.SetUserTwoFactorEnabled(ctx, userID, false); err != nil {
		return err
	}

	// Delete trusted devices
	if err := uc.Repo.DeleteTrustedDevicesByUserID(ctx, userID); err != nil {
		return err
	}

	return nil
}
