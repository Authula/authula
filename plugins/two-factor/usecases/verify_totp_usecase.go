package usecases

import (
	"context"
	"time"

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

	// Publish verified event
	publishEvent(uc.EventBus, uc.Logger, constants.EventTwoFactorVerified, userID)

	return result, nil
}
