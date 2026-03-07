package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type enableUseCase struct {
	UserService       rootservices.UserService
	AccountService    rootservices.AccountService
	PasswordService   rootservices.PasswordService
	TokenService      rootservices.TokenService
	TOTPService       *services.TOTPService
	BackupCodeService *services.BackupCodeService
	TwoFactorRepo     *repository.TwoFactorRepository
	Config            *types.TwoFactorPluginConfig
	EventBus          models.EventBus
	Logger            models.Logger
}

func NewEnableUseCase(
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	tokenService rootservices.TokenService,
	totpService *services.TOTPService,
	backupCodeService *services.BackupCodeService,
	twoFactorRepo *repository.TwoFactorRepository,
	config *types.TwoFactorPluginConfig,
	eventBus models.EventBus,
	logger models.Logger,
) EnableUseCase {
	return &enableUseCase{
		UserService:       userService,
		AccountService:    accountService,
		PasswordService:   passwordService,
		TokenService:      tokenService,
		TOTPService:       totpService,
		BackupCodeService: backupCodeService,
		TwoFactorRepo:     twoFactorRepo,
		Config:            config,
		EventBus:          eventBus,
		Logger:            logger,
	}
}

func (uc *enableUseCase) Enable(ctx context.Context, userID, password, issuer string) (*types.EnableResult, error) {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return nil, err
	}

	// Check if 2FA is already enabled
	existing, err := uc.TwoFactorRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, constants.ErrTwoFactorAlreadyEnabled
	}

	// Fetch user to get email for TOTP URI
	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, constants.ErrUserNotFound
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
	_ = uc.TwoFactorRepo.DeleteByUserID(ctx, userID)

	_, err = uc.TwoFactorRepo.Create(ctx, userID, encryptedSecret, encryptedBackup)
	if err != nil {
		return nil, err
	}

	// If SkipVerificationOnEnable, mark 2FA as enabled immediately
	if uc.Config.SkipVerificationOnEnable {
		if err := uc.TwoFactorRepo.SetEnabled(ctx, userID, true); err != nil {
			return nil, err
		}
	}

	// Build TOTP URI
	if issuer == "" {
		issuer = uc.Config.Issuer
	}
	totpURI := uc.TOTPService.BuildURI(secret, issuer, user.Email)

	// Publish enabled event
	payload, err := json.Marshal(map[string]string{"userID": userID})
	if err != nil {
		uc.Logger.Error(err.Error())
	} else {
		util.PublishEventAsync(
			uc.EventBus,
			uc.Logger,
			models.Event{
				ID:        util.GenerateUUID(),
				Type:      constants.EventTwoFactorEnabled,
				Payload:   payload,
				Metadata:  nil,
				Timestamp: time.Now().UTC(),
			},
		)
	}

	return &types.EnableResult{
		TotpURI:     totpURI,
		BackupCodes: backupCodes,
	}, nil
}
