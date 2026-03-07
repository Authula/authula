package usecases

import (
	"context"
	"encoding/json"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type generateBackupCodesUseCase struct {
	AccountService    rootservices.AccountService
	PasswordService   rootservices.PasswordService
	TokenService      rootservices.TokenService
	BackupCodeService *services.BackupCodeService
	TwoFactorRepo     *repository.TwoFactorRepository
}

func NewGenerateBackupCodesUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	tokenService rootservices.TokenService,
	backupCodeService *services.BackupCodeService,
	twoFactorRepo *repository.TwoFactorRepository,
) GenerateBackupCodesUseCase {
	return &generateBackupCodesUseCase{
		AccountService:    accountService,
		PasswordService:   passwordService,
		TokenService:      tokenService,
		BackupCodeService: backupCodeService,
		TwoFactorRepo:     twoFactorRepo,
	}
}

func (uc *generateBackupCodesUseCase) Generate(ctx context.Context, userID, password string) ([]string, error) {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return nil, err
	}

	// Check that 2FA is enabled
	existing, err := uc.TwoFactorRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, constants.ErrTwoFactorNotEnabled
	}

	// Generate new backup codes
	codes, err := uc.BackupCodeService.Generate()
	if err != nil {
		return nil, err
	}

	// Encrypt and update in DB
	codesJSON, err := json.Marshal(codes)
	if err != nil {
		return nil, err
	}
	encrypted, err := uc.TokenService.Encrypt(string(codesJSON))
	if err != nil {
		return nil, err
	}
	if err := uc.TwoFactorRepo.UpdateBackupCodes(ctx, userID, encrypted); err != nil {
		return nil, err
	}

	return codes, nil
}
