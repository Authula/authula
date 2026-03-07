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

type verifyTOTPUseCase struct {
	TokenService        rootservices.TokenService
	SessionService      rootservices.SessionService
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TOTPService         *services.TOTPService
	TwoFactorRepo       *repository.TwoFactorRepository
	GlobalConfig        *models.Config
	Config              *types.TwoFactorPluginConfig
	EventBus            models.EventBus
	Logger              models.Logger
}

func NewVerifyTOTPUseCase(
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	userService rootservices.UserService,
	verificationService rootservices.VerificationService,
	totpService *services.TOTPService,
	twoFactorRepo *repository.TwoFactorRepository,
	globalConfig *models.Config,
	config *types.TwoFactorPluginConfig,
	eventBus models.EventBus,
	logger models.Logger,
) VerifyTOTPUseCase {
	return &verifyTOTPUseCase{
		TokenService:        tokenService,
		SessionService:      sessionService,
		UserService:         userService,
		VerificationService: verificationService,
		TOTPService:         totpService,
		TwoFactorRepo:       twoFactorRepo,
		GlobalConfig:        globalConfig,
		Config:              config,
		EventBus:            eventBus,
		Logger:              logger,
	}
}

func (uc *verifyTOTPUseCase) Verify(ctx context.Context, pendingToken, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error) {
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

	// Decrypt secret
	secret, err := uc.TokenService.Decrypt(record.Secret)
	if err != nil {
		return nil, err
	}

	// Validate TOTP code
	if !uc.TOTPService.ValidateCode(secret, code, time.Now().UTC()) {
		return nil, constants.ErrInvalidTOTPCode
	}

	// If this is the first successful verification (SkipVerificationOnEnable was false),
	// enable 2FA on the two_factor record
	if !record.Enabled {
		if err := uc.TwoFactorRepo.SetEnabled(ctx, userID, true); err != nil {
			return nil, err
		}
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

	// Publish verified event
	uc.publishEvent(constants.EventTwoFactorVerified, userID)

	return result, nil
}

// resolvePendingToken hashes the raw token, looks up the verification record,
// checks expiry, and returns the userID and verification ID.
func (uc *verifyTOTPUseCase) resolvePendingToken(ctx context.Context, rawToken string) (userID, verificationID string, err error) {
	hashedToken := uc.TokenService.Hash(rawToken)
	verification, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return "", "", constants.ErrInvalidPendingToken
	}
	if verification == nil || verification.UserID == nil {
		return "", "", constants.ErrInvalidPendingToken
	}
	if verification.Type != models.TypeTwoFactorPendingAuth {
		return "", "", constants.ErrInvalidPendingToken
	}
	if verification.ExpiresAt.Before(time.Now().UTC()) {
		return "", "", constants.ErrInvalidPendingToken
	}
	return *verification.UserID, verification.ID, nil
}

func (uc *verifyTOTPUseCase) publishEvent(eventType, userID string) {
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
