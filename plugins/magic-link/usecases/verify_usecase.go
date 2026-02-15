package usecases

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type VerifyUseCaseImpl struct {
	GlobalConfig        *models.Config
	PluginConfig        *types.MagicLinkPluginConfig
	Logger              models.Logger
	UserService         rootservices.UserService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
}

func (uc *VerifyUseCaseImpl) Verify(
	ctx context.Context,
	token string,
	ipAddress *string,
	userAgent *string,
) (string, error) {
	hashedToken := uc.TokenService.Hash(token)
	verification, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return "", err
	}

	if verification == nil || uc.VerificationService.IsExpired(verification) {
		return "", fmt.Errorf("invalid or expired token")
	}

	if verification.Type != models.TypeMagicLinkSignInRequest {
		return "", fmt.Errorf("invalid token type")
	}

	user, err := uc.UserService.GetByID(ctx, *verification.UserID)
	if err != nil {
		return "", err
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	if !user.EmailVerified {
		if err := uc.UserService.UpdateFields(ctx, user.ID, map[string]any{
			"email_verified": true,
		}); err != nil {
			return "", err
		}
	}

	if err := uc.VerificationService.Delete(ctx, verification.ID); err != nil {
		return "", err
	}

	newToken, err := uc.TokenService.Generate()
	if err != nil {
		return "", err
	}

	newHashedToken := uc.TokenService.Hash(newToken)
	_, err = uc.VerificationService.Create(
		ctx,
		user.ID,
		newHashedToken,
		models.TypeMagicLinkExchangeCode,
		user.Email,
		uc.PluginConfig.ExpiresIn,
	)
	if err != nil {
		return "", err
	}

	return newToken, nil
}
