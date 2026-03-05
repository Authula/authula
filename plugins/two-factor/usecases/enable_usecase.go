package usecases

import (
	"context"
	"encoding/json"

	twofactor "github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type enableUseCase struct {
	AccountService    rootservices.AccountService
	PasswordService   rootservices.PasswordService
	TokenService      rootservices.TokenService
	TOTPService       *services.TOTPService
	BackupCodeService *services.BackupCodeService
	Repo              *twofactor.TwoFactorRepository
	Config            *types.TwoFactorPluginConfig
}

func NewEnableUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	tokenService rootservices.TokenService,
	totpService *services.TOTPService,
	backupCodeService *services.BackupCodeService,
	repo *twofactor.TwoFactorRepository,
	config *types.TwoFactorPluginConfig,
) EnableUseCase {
	return &enableUseCase{
		AccountService:    accountService,
		PasswordService:   passwordService,
		TokenService:      tokenService,
		TOTPService:       totpService,
		BackupCodeService: backupCodeService,
		Repo:              repo,
		Config:            config,
	}
}

func (uc *enableUseCase) Enable(ctx context.Context, userID, password, issuer, email string) (*types.EnableResult, error) {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return nil, err
	}

	// Check if 2FA is already enabled
	existing, err := uc.Repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, constants.ErrTwoFactorAlreadyEnabled
	}

	// Generate TOTP secret
	secret, err := uc.TOTPService.GenerateSecret()
	if err != nil {
		return nil, err
	}

	// Generate backup codes
	backupCodes, err := uc.BackupCodeService.Generate()
	if err != nil {
		return nil, err
	}

	// Encrypt secret
	encryptedSecret, err := uc.TokenService.Encrypt(secret)
	if err != nil {
		return nil, err
	}

	// Encrypt backup codes (as JSON array)
	backupJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return nil, err
	}
	encryptedBackup, err := uc.TokenService.Encrypt(string(backupJSON))
	if err != nil {
		return nil, err
	}

	// Delete any existing record (in case of stale data) and create new
	_ = uc.Repo.DeleteByUserID(ctx, userID)

	_, err = uc.Repo.Create(ctx, userID, encryptedSecret, encryptedBackup)
	if err != nil {
		return nil, err
	}

	// If SkipVerificationOnEnable, mark user as 2FA-enabled immediately
	if uc.Config.SkipVerificationOnEnable {
		if err := uc.Repo.SetUserTwoFactorEnabled(ctx, userID, true); err != nil {
			return nil, err
		}
	}

	// Build TOTP URI
	if issuer == "" {
		issuer = uc.Config.Issuer
	}
	totpURI := uc.TOTPService.BuildURI(secret, issuer, email)

	return &types.EnableResult{
		TotpURI:     totpURI,
		BackupCodes: backupCodes,
	}, nil
}
