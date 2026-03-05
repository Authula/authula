package usecases

import (
	"context"
	"time"

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
	repo *repository.TwoFactorRepository,
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

	expiresAt := time.Now().Add(config.TrustedDeviceDuration)
	_, err = repo.CreateTrustedDevice(ctx, userID, hashedToken, ua, expiresAt)
	if err != nil {
		return "", err
	}

	return deviceToken, nil // Return raw token for cookie
}
