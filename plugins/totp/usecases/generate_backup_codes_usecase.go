package usecases

import (
	"context"
	"encoding/json"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
)

type GenerateBackupCodesUseCase struct {
	BackupCodeService *services.BackupCodeService
	TOTPRepo          TOTPRepository
}

func NewGenerateBackupCodesUseCase(
	backupCodeService *services.BackupCodeService,
	totpRepo TOTPRepository,
) *GenerateBackupCodesUseCase {
	return &GenerateBackupCodesUseCase{
		BackupCodeService: backupCodeService,
		TOTPRepo:          totpRepo,
	}
}

func (uc *GenerateBackupCodesUseCase) Generate(ctx context.Context, userID string) ([]string, error) {
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
