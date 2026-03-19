package usecases

import (
	"context"
	"encoding/json"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type EnableUseCase struct {
	Config            *types.TOTPPluginConfig
	Logger            models.Logger
	EventBus          models.EventBus
	UserService       rootservices.UserService
	TokenService      rootservices.TokenService
	Verification      rootservices.VerificationService
	TOTPService       *services.TOTPService
	BackupCodeService *services.BackupCodeService
	TOTPRepo          TOTPRepository
}

func NewEnableUseCase(
	config *types.TOTPPluginConfig,
	logger models.Logger,
	eventBus models.EventBus,
	userService rootservices.UserService,
	tokenService rootservices.TokenService,
	verificationService rootservices.VerificationService,
	totpService *services.TOTPService,
	backupCodeService *services.BackupCodeService,
	totpRepo TOTPRepository,
) *EnableUseCase {
	return &EnableUseCase{
		Config:            config,
		Logger:            logger,
		EventBus:          eventBus,
		UserService:       userService,
		TokenService:      tokenService,
		Verification:      verificationService,
		TOTPService:       totpService,
		BackupCodeService: backupCodeService,
		TOTPRepo:          totpRepo,
	}
}

func (uc *EnableUseCase) Enable(ctx context.Context, userID string, issuer string) (*types.EnableResult, error) {
	existing, err := uc.TOTPRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Enabled {
		return nil, constants.ErrTOTPAlreadyEnabled
	}

	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, constants.ErrUserNotFound
	}

	secret, err := uc.TOTPService.GenerateSecret()
	if err != nil {
		return nil, err
	}
	encryptedSecret, err := uc.TokenService.Encrypt(secret)
	if err != nil {
		return nil, err
	}

	backupCodes, err := uc.BackupCodeService.Generate()
	if err != nil {
		return nil, err
	}
	hashedCodes, err := uc.BackupCodeService.HashCodes(backupCodes)
	if err != nil {
		return nil, err
	}
	hashedJSON, err := json.Marshal(hashedCodes)
	if err != nil {
		return nil, err
	}

	if err := uc.TOTPRepo.DeleteByUserID(ctx, userID); err != nil {
		return nil, err
	}

	_, err = uc.TOTPRepo.Create(ctx, userID, encryptedSecret, string(hashedJSON))
	if err != nil {
		return nil, err
	}

	result := &types.EnableResult{}

	if uc.Config.SkipVerificationOnEnable {
		if err := uc.TOTPRepo.SetEnabled(ctx, userID, true); err != nil {
			return nil, err
		}
	} else {
		token, err := uc.TokenService.Generate()
		if err != nil {
			return nil, err
		}

		hashedToken := uc.TokenService.Hash(token)
		_, err = uc.Verification.Create(
			ctx,
			userID,
			hashedToken,
			models.TypeTOTPPendingAuth,
			userID,
			uc.Config.PendingTokenExpiry,
		)
		if err != nil {
			return nil, err
		}

		result.PendingToken = token
	}

	totpURI := uc.TOTPService.BuildURI(secret, issuer, user.Email)

	publishEvent(uc.EventBus, uc.Logger, constants.EventTOTPEnabled, userID)

	result.TotpURI = totpURI
	result.BackupCodes = backupCodes

	return result, nil
}
