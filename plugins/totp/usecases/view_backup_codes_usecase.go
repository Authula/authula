package usecases

import (
	"context"
	"encoding/json"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type ViewBackupCodesUseCase struct {
	AccountService  rootservices.AccountService
	PasswordService rootservices.PasswordService
	TOTPRepo        TOTPReadRepository
}

func NewViewBackupCodesUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	totpRepo TOTPReadRepository,
) *ViewBackupCodesUseCase {
	return &ViewBackupCodesUseCase{
		AccountService:  accountService,
		PasswordService: passwordService,
		TOTPRepo:        totpRepo,
	}
}

func (uc *ViewBackupCodesUseCase) View(ctx context.Context, userID, password string) (int, error) {
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return 0, err
	}

	record, err := uc.TOTPRepo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if record == nil {
		return 0, constants.ErrTOTPNotEnabled
	}

	var hashedCodes []string
	if err := json.Unmarshal([]byte(record.BackupCodes), &hashedCodes); err != nil {
		return 0, err
	}

	return len(hashedCodes), nil
}
