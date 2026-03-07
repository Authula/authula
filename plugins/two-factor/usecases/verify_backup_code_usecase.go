package usecases

import (
	"context"
	"encoding/json"
	"errors"
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
	userID, verificationID, err := uc.resolvePendingToken(ctx, pendingToken)
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
	token, err := uc.TokenService.Generate()
	if err != nil {
		return nil, err
	}
	hashedToken := uc.TokenService.Hash(token)

	session, err := uc.SessionService.Create(ctx, userID, hashedToken, ipAddress, userAgent, uc.GlobalConfig.Session.ExpiresIn)
	if err != nil {
		return nil, err
	}

	// Delete the pending verification
	_ = uc.VerificationService.Delete(ctx, verificationID)

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
		uc.publishEvent(constants.EventTwoFactorDeviceTrusted, userID)
	}

	// Publish backup code used event
	uc.publishEvent(constants.EventTwoFactorBackupUsed, userID)

	return result, nil
}

// resolvePendingToken hashes the raw token, looks up the verification record,
// checks expiry, and returns the userID and verification ID.
func (uc *verifyBackupCodeUseCase) resolvePendingToken(ctx context.Context, rawToken string) (userID, verificationID string, err error) {
	hashedToken := uc.TokenService.Hash(rawToken)
	verification, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return "", "", errors.New(err.Error())
	}
	if verification == nil || verification.UserID == nil {
		return "", "", constants.ErrInvalidPendingToken
	}
	if verification.Type != models.TypeTwoFactorPendingAuth {
		return "", "", constants.ErrInvalidVerificationType
	}
	if verification.ExpiresAt.Before(time.Now().UTC()) {
		return "", "", constants.ErrPendingTokenExpired
	}
	return *verification.UserID, verification.ID, nil
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
