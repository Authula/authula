package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	twofactor "github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type verifyBackupCodeUseCase struct {
	TokenService      rootservices.TokenService
	SessionService    rootservices.SessionService
	UserService       rootservices.UserService
	BackupCodeService *services.BackupCodeService
	Repo              *twofactor.TwoFactorRepository
	GlobalConfig      *models.Config
	Config            *types.TwoFactorPluginConfig
}

func NewVerifyBackupCodeUseCase(
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	userService rootservices.UserService,
	backupCodeService *services.BackupCodeService,
	repo *twofactor.TwoFactorRepository,
	globalConfig *models.Config,
	config *types.TwoFactorPluginConfig,
) VerifyBackupCodeUseCase {
	return &verifyBackupCodeUseCase{
		TokenService:      tokenService,
		SessionService:    sessionService,
		UserService:       userService,
		BackupCodeService: backupCodeService,
		Repo:              repo,
		GlobalConfig:      globalConfig,
		Config:            config,
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
		deviceToken, err := uc.createTrustedDevice(ctx, userID, userAgent)
		if err != nil {
			return nil, err
		}
		result.TrustedDeviceToken = deviceToken
	}

	return result, nil
}

func (uc *verifyBackupCodeUseCase) createTrustedDevice(ctx context.Context, userID string, userAgent *string) (string, error) {
	token, err := uc.TokenService.Generate()
	if err != nil {
		return "", err
	}

	ua := ""
	if userAgent != nil {
		ua = *userAgent
	}

	expiresAt := time.Now().Add(uc.Config.TrustedDeviceDuration)
	_, err = uc.Repo.CreateTrustedDevice(ctx, userID, token, ua, expiresAt)
	if err != nil {
		return "", err
	}

	return token, nil
}
