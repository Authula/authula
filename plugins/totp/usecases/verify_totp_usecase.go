package usecases

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type VerifyTOTPUseCase struct {
	GlobalConfig        *models.Config
	Config              *types.TOTPPluginConfig
	Logger              models.Logger
	EventBus            models.EventBus
	TokenService        rootservices.TokenService
	SessionService      rootservices.SessionService
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TOTPService         *services.TOTPService
	TOTPRepo            TOTPRepository
}

func NewVerifyTOTPUseCase(
	globalConfig *models.Config,
	config *types.TOTPPluginConfig,
	logger models.Logger,
	eventBus models.EventBus,
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	userService rootservices.UserService,
	verificationService rootservices.VerificationService,
	totpService *services.TOTPService,
	totpRepo TOTPRepository,
) *VerifyTOTPUseCase {
	return &VerifyTOTPUseCase{
		GlobalConfig:        globalConfig,
		Config:              config,
		Logger:              logger,
		EventBus:            eventBus,
		TokenService:        tokenService,
		SessionService:      sessionService,
		UserService:         userService,
		VerificationService: verificationService,
		TOTPService:         totpService,
		TOTPRepo:            totpRepo,
	}
}

func (uc *VerifyTOTPUseCase) Verify(ctx context.Context, pendingToken, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error) {
	userID, verificationID, err := resolvePendingToken(ctx, uc.TokenService, uc.VerificationService, pendingToken)
	if err != nil {
		return nil, err
	}

	record, err := uc.TOTPRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, constants.ErrTOTPNotEnabled
	}

	secret, err := uc.TokenService.Decrypt(record.Secret)
	if err != nil {
		return nil, err
	}

	if !uc.TOTPService.ValidateCode(secret, code, time.Now().UTC()) {
		return nil, constants.ErrInvalidTOTPCode
	}

	if !record.Enabled {
		if err := uc.TOTPRepo.SetEnabled(ctx, userID, true); err != nil {
			return nil, err
		}
	}

	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, constants.ErrUserNotFound
	}

	session, token, err := createSessionForUser(ctx, uc.TokenService, uc.SessionService, uc.VerificationService, uc.GlobalConfig, userID, verificationID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	result := &types.VerifyResult{
		User:         user,
		Session:      session,
		SessionToken: token,
	}

	if trustDevice {
		deviceToken, err := createTrustedDevice(ctx, uc.TokenService, uc.TOTPRepo, uc.Config, userID, userAgent)
		if err != nil {
			return nil, err
		}
		result.TrustedDeviceToken = deviceToken

		publishEvent(uc.EventBus, uc.Logger, constants.EventTOTPDeviceTrusted, userID)
	}

	publishEvent(uc.EventBus, uc.Logger, constants.EventTOTPVerified, userID)

	return result, nil
}
