package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

func createTrustedDevice(
	ctx context.Context,
	config *types.TOTPPluginConfig,
	userID string,
	userAgent *string,
	tokenService rootservices.TokenService,
	totpRepo TOTPCreateRepository,
) (string, error) {
	deviceToken, err := tokenService.Generate()
	if err != nil {
		return "", err
	}
	hashedToken := tokenService.Hash(deviceToken)

	assignedUserAgent := ""
	if userAgent != nil {
		assignedUserAgent = *userAgent
	}

	expiresAt := time.Now().UTC().Add(config.TrustedDeviceDuration)
	_, err = totpRepo.CreateTrustedDevice(ctx, userID, hashedToken, assignedUserAgent, expiresAt)
	if err != nil {
		return "", err
	}

	return deviceToken, nil
}

func resolvePendingToken(
	ctx context.Context,
	tokenService rootservices.TokenService,
	verificationService rootservices.VerificationService,
	rawToken string,
) (userID, verificationID string, err error) {
	hashedToken := tokenService.Hash(rawToken)
	verification, err := verificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return "", "", err
	}
	if verification == nil || verification.UserID == nil {
		return "", "", constants.ErrInvalidPendingToken
	}
	if verification.Type != models.TypeTOTPPendingAuth {
		return "", "", constants.ErrInvalidVerificationType
	}
	if verificationService.IsExpired(verification) {
		return "", "", constants.ErrPendingTokenExpired
	}
	return *verification.UserID, verification.ID, nil
}

func publishEvent(eventBus models.EventBus, logger models.Logger, eventType, userID string) {
	payload, err := json.Marshal(map[string]string{"userID": userID})
	if err != nil {
		logger.Error(err.Error())
		return
	}
	util.PublishEventAsync(
		eventBus,
		logger,
		models.Event{
			ID:        util.GenerateUUID(),
			Type:      eventType,
			Payload:   payload,
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}

func createSessionForUser(
	ctx context.Context,
	globalConfig *models.Config,
	userID string,
	verificationID string,
	ipAddress *string,
	userAgent *string,
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	verificationService rootservices.VerificationService,
) (*models.Session, string, error) {
	token, err := tokenService.Generate()
	if err != nil {
		return nil, "", err
	}
	hashedToken := tokenService.Hash(token)

	session, err := sessionService.Create(ctx, userID, hashedToken, ipAddress, userAgent, globalConfig.Session.ExpiresIn)
	if err != nil {
		return nil, "", err
	}

	if err := verificationService.Delete(ctx, verificationID); err != nil {
		return nil, "", err
	}

	return session, token, nil
}
