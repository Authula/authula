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
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

func verifyPassword(
	ctx context.Context,
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	userID, password string,
) error {
	if password == "" {
		return constants.ErrPasswordRequired
	}
	account, err := accountService.GetByUserIDAndProvider(ctx, userID, models.AuthProviderEmail.String())
	if err != nil || account == nil {
		return constants.ErrAccountNotFound
	}
	if account.Password == nil || !passwordService.Verify(password, *account.Password) {
		return constants.ErrInvalidPassword
	}
	return nil
}

// createTrustedDevice generates a trusted device token, hashes it for storage,
// and returns the raw token for use in the cookie.
func createTrustedDevice(
	ctx context.Context,
	tokenService rootservices.TokenService,
	twoFactorRepo *repository.TwoFactorRepository,
	config *types.TwoFactorPluginConfig,
	userID string,
	userAgent *string,
) (string, error) {
	deviceToken, err := tokenService.Generate()
	if err != nil {
		return "", err
	}
	hashedToken := tokenService.Hash(deviceToken)

	ua := ""
	if userAgent != nil {
		ua = *userAgent
	}

	expiresAt := time.Now().UTC().Add(config.TrustedDeviceDuration)
	_, err = twoFactorRepo.CreateTrustedDevice(ctx, userID, hashedToken, ua, expiresAt)
	if err != nil {
		return "", err
	}

	return deviceToken, nil // Return raw token for cookie
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
	tokenService rootservices.TokenService,
	sessionService rootservices.SessionService,
	verificationService rootservices.VerificationService,
	globalConfig *models.Config,
	userID, verificationID string,
	ipAddress, userAgent *string,
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

	// Delete the pending verification
	_ = verificationService.Delete(ctx, verificationID)

	return session, token, nil
}
