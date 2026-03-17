package usecases

import (
	"context"
	"encoding/json"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type GenerateBackupCodesUseCase struct {
	AccountService    rootservices.AccountService
	PasswordService   rootservices.PasswordService
	BackupCodeService *services.BackupCodeService
	TOTPRepo          TOTPRepository
}

func NewGenerateBackupCodesUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	backupCodeService *services.BackupCodeService,
	totpRepo TOTPRepository,
) *GenerateBackupCodesUseCase {
	return &GenerateBackupCodesUseCase{
		AccountService:    accountService,
		PasswordService:   passwordService,
		BackupCodeService: backupCodeService,
		TOTPRepo:          totpRepo,
	}
}

func (uc *GenerateBackupCodesUseCase) Generate(ctx context.Context, userID string, password string) ([]string, error) {
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return nil, err
	}

	existing, err := uc.TOTPRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, constants.ErrTOTPNotEnabled
	}

	codes, err := uc.BackupCodeService.Generate()
	if err != nil {
		return nil, err
	}

	hashedCodes, err := uc.BackupCodeService.HashCodes(codes)
	if err != nil {
		return nil, err
	}
	hashedJSON, err := json.Marshal(hashedCodes)
	if err != nil {
		return nil, err
	}
	if err := uc.TOTPRepo.UpdateBackupCodes(ctx, userID, string(hashedJSON)); err != nil {
		return nil, err
	}

	return codes, nil
}
