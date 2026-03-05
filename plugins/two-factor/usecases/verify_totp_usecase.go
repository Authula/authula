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
	TokenService   rootservices.TokenService
	SessionService rootservices.SessionService
	UserService    rootservices.UserService
	TOTPService    *services.TOTPService
	Repo           *repository.TwoFactorRepository
	GlobalConfig   *models.Config
	Config         *types.TwoFactorPluginConfig
	EventBus       models.EventBus
	Logger         models.Logger
}

func NewVerifyTOTPUseCase(
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	userService rootservices.UserService,
	totpService *services.TOTPService,
	repo *repository.TwoFactorRepository,
	globalConfig *models.Config,
	config *types.TwoFactorPluginConfig,
	eventBus models.EventBus,
	logger models.Logger,
) VerifyTOTPUseCase {
	return &verifyTOTPUseCase{
		TokenService:   tokenService,
		SessionService: sessionService,
		UserService:    userService,
		TOTPService:    totpService,
		Repo:           repo,
		GlobalConfig:   globalConfig,
		Config:         config,
		EventBus:       eventBus,
		Logger:         logger,
	}
}

func (uc *verifyTOTPUseCase) Verify(ctx context.Context, userID, code string, trustDevice bool, ipAddress, userAgent *string) (*types.VerifyResult, error) {
	// Get two-factor record
	record, err := uc.Repo.GetByUserID(ctx, userID)
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
	if !uc.TOTPService.ValidateCode(secret, code, time.Now()) {
		return nil, constants.ErrInvalidTOTPCode
	}

	// If this is the first successful verification (SkipVerificationOnEnable was false),
	// enable 2FA on the user
	user, err := uc.UserService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, constants.ErrUserNotFound
	}

	if user.TwoFactorEnabled == nil || !*user.TwoFactorEnabled {
		if err := uc.Repo.SetUserTwoFactorEnabled(ctx, userID, true); err != nil {
			return nil, err
		}
		enabled := true
		user.TwoFactorEnabled = &enabled
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

	// Publish verified event
	uc.publishEvent(constants.EventTwoFactorVerified, userID)

	return result, nil
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
