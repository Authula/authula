package usecases

import (
	"context"
	"encoding/json"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/repository"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type verifyBackupCodeUseCase struct {
	TokenService        rootservices.TokenService
	SessionService      rootservices.SessionService
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	BackupCodeService   *services.BackupCodeService
	TwoFactorRepo       *repository.TwoFactorRepository
	GlobalConfig        *models.Config
	Config              *types.TwoFactorPluginConfig
	EventBus            models.EventBus
	Logger              models.Logger
}

func NewVerifyBackupCodeUseCase(
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	userService rootservices.UserService,
	verificationService rootservices.VerificationService,
	backupCodeService *services.BackupCodeService,
	twoFactorRepo *repository.TwoFactorRepository,
	globalConfig *models.Config,
	config *types.TwoFactorPluginConfig,
	eventBus models.EventBus,
	logger models.Logger,
) VerifyBackupCodeUseCase {
	return &verifyBackupCodeUseCase{
		TokenService:        tokenService,
		SessionService:      sessionService,
		UserService:         userService,
		VerificationService: verificationService,
		BackupCodeService:   backupCodeService,
		TwoFactorRepo:       twoFactorRepo,
		GlobalConfig:        globalConfig,
		Config:              config,
		EventBus:            eventBus,
		Logger:              logger,
	}
}

func (uc *verifyBackupCodeUseCase) Verify(ctx context.Context, pendingToken, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error) {
	// Resolve userID from pending token
	userID, verificationID, err := resolvePendingToken(ctx, uc.TokenService, uc.VerificationService, pendingToken)
	if err != nil {
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

	// Unmarshal hashed backup codes
	var hashedCodes []string
	if err := json.Unmarshal([]byte(record.BackupCodes), &hashedCodes); err != nil {
		return nil, err
	}

	// Verify and consume the backup code
	remaining, valid := uc.BackupCodeService.VerifyAndConsume(hashedCodes, code)
	if !valid {
		return nil, constants.ErrInvalidBackupCode
	}

	// Update remaining codes in DB
	remainingJSON, err := json.Marshal(remaining)
	if err != nil {
		return nil, err
	}
	if err := uc.TwoFactorRepo.UpdateBackupCodes(ctx, userID, string(remainingJSON)); err != nil {
		return nil, err
	}

	// Get user
	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, constants.ErrUserNotFound
	}

	// Create session
	session, token, err := createSessionForUser(ctx, uc.TokenService, uc.SessionService, uc.VerificationService, uc.GlobalConfig, userID, verificationID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	result := &types.VerifyResult{
		User:                  user,
		Session:               session,
		SessionToken:          token,
		TrustedDeviceDuration: uc.Config.TrustedDeviceDuration,
		SecureCookie:          uc.Config.SecureCookie,
		SameSite:              types.ParseSameSite(uc.Config.SameSite),
	}

	// Optionally trust device
	if trustDevice {
		deviceToken, err := createTrustedDevice(ctx, uc.TokenService, uc.TwoFactorRepo, uc.Config, userID, userAgent)
		if err != nil {
			return nil, err
		}
		result.TrustedDeviceToken = deviceToken

		// Publish device trusted event
		publishEvent(uc.EventBus, uc.Logger, constants.EventTwoFactorDeviceTrusted, userID)
	}

	// Publish backup code used event
	publishEvent(uc.EventBus, uc.Logger, constants.EventTwoFactorBackupUsed, userID)

	return result, nil
}
