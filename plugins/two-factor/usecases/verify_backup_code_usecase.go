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

type verifyBackupCodeUseCase struct {
	TokenService      rootservices.TokenService
	SessionService    rootservices.SessionService
	UserService       rootservices.UserService
	BackupCodeService *services.BackupCodeService
	Repo              *repository.TwoFactorRepository
	GlobalConfig      *models.Config
	Config            *types.TwoFactorPluginConfig
	EventBus          models.EventBus
	Logger            models.Logger
}

func NewVerifyBackupCodeUseCase(
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	userService rootservices.UserService,
	backupCodeService *services.BackupCodeService,
	repo *repository.TwoFactorRepository,
	globalConfig *models.Config,
	config *types.TwoFactorPluginConfig,
	eventBus models.EventBus,
	logger models.Logger,
) VerifyBackupCodeUseCase {
	return &verifyBackupCodeUseCase{
		TokenService:      tokenService,
		SessionService:    sessionService,
		UserService:       userService,
		BackupCodeService: backupCodeService,
		Repo:              repo,
		GlobalConfig:      globalConfig,
		Config:            config,
		EventBus:          eventBus,
		Logger:            logger,
	}
}

func (uc *verifyBackupCodeUseCase) Verify(ctx context.Context, userID, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error) {
	// Get two-factor record
	record, err := uc.Repo.GetByUserID(ctx, userID)
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

	// Verify and consume the backup code
	remaining, valid := uc.BackupCodeService.VerifyAndConsume(codes, code)
	if !valid {
		return nil, constants.ErrInvalidBackupCode
	}

	// Update remaining codes in DB
	remainingJSON, err := json.Marshal(remaining)
	if err != nil {
		return nil, err
	}
	encryptedRemaining, err := uc.TokenService.Encrypt(string(remainingJSON))
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.UpdateBackupCodes(ctx, userID, encryptedRemaining); err != nil {
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
	token, err := uc.TokenService.Generate()
	if err != nil {
		return nil, err
	}
	hashedToken := uc.TokenService.Hash(token)

	session, err := uc.SessionService.Create(ctx, userID, hashedToken, ipAddress, userAgent, uc.GlobalConfig.Session.ExpiresIn)
	if err != nil {
		return nil, err
	}

	result := &types.VerifyResult{
		User:         user,
		Session:      session,
		SessionToken: token,
	}

	// Optionally trust device
	if trustDevice {
		deviceToken, err := createTrustedDevice(ctx, uc.TokenService, uc.Repo, uc.Config, userID, userAgent)
		if err != nil {
			return nil, err
		}
		result.TrustedDeviceToken = deviceToken

		// Publish device trusted event
		uc.publishEvent(constants.EventTwoFactorDeviceTrusted, userID)
	}

	// Publish backup code used event
	uc.publishEvent(constants.EventTwoFactorBackupUsed, userID)

	return result, nil
}

func (uc *verifyBackupCodeUseCase) publishEvent(eventType, userID string) {
	payload, err := json.Marshal(map[string]string{"userID": userID})
	if err != nil {
		uc.Logger.Error(err.Error())
		return
	}
	util.PublishEventAsync(
		uc.EventBus,
		uc.Logger,
		models.Event{
			ID:        util.GenerateUUID(),
			Type:      eventType,
			Payload:   payload,
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}
