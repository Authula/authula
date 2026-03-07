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
	TokenService    rootservices.TokenService
	TwoFactorRepo   *repository.TwoFactorRepository
}

func NewViewBackupCodesUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	tokenService rootservices.TokenService,
	twoFactorRepo *repository.TwoFactorRepository,
) ViewBackupCodesUseCase {
	return &viewBackupCodesUseCase{
		AccountService:  accountService,
		PasswordService: passwordService,
		TokenService:    tokenService,
		TwoFactorRepo:   twoFactorRepo,
	}
}

func (uc *viewBackupCodesUseCase) View(ctx context.Context, userID, password string) ([]string, error) {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return nil, err
	}

	// Get two-factor record
	record, err := uc.TwoFactorRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, constants.ErrTwoFactorNotEnabled
	}

	// Decrypt backup codes
	decryptedJSON, err := uc.TokenService.Decrypt(record.BackupCodes)
	if err != nil {
		return nil, err
	}

	var codes []string
	if err := json.Unmarshal([]byte(decryptedJSON), &codes); err != nil {
		return nil, err
	}

	return codes, nil
}
