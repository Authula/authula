package usecases

import (
	"context"
	"encoding/json"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/repository"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type viewBackupCodesUseCase struct {
	AccountService  rootservices.AccountService
	PasswordService rootservices.PasswordService
	TwoFactorRepo   *repository.TwoFactorRepository
}

func NewViewBackupCodesUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	twoFactorRepo *repository.TwoFactorRepository,
) ViewBackupCodesUseCase {
	return &viewBackupCodesUseCase{
		AccountService:  accountService,
		PasswordService: passwordService,
		TwoFactorRepo:   twoFactorRepo,
	}
}

func (uc *viewBackupCodesUseCase) View(ctx context.Context, userID, password string) (int, error) {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return 0, err
	}

	// Get two-factor record
	record, err := uc.TwoFactorRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if record == nil {
		return 0, constants.ErrTwoFactorNotEnabled
	}

	// Unmarshal hashed backup codes to count them
	var hashedCodes []string
	if err := json.Unmarshal([]byte(record.BackupCodes), &hashedCodes); err != nil {
		return 0, err
	}

	return len(hashedCodes), nil
}
