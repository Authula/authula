package usecases

import (
	"context"
	"fmt"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
	rootservices "github.com/Authula/authula/services"
)

type ExchangeUseCaseImpl struct {
	GlobalConfig        *models.Config
	PluginConfig        *types.MagicLinkPluginConfig
	Logger              models.Logger
	UserService         rootservices.UserService
	AccountService      rootservices.AccountService
	SessionService      rootservices.SessionService
	VerificationService rootservices.VerificationService
	TokenService        rootservices.TokenService
}

func (uc *ExchangeUseCaseImpl) Exchange(
	ctx context.Context,
	token string,
	ipAddress *string,
	userAgent *string,
) (*types.MagicLinkExchangeResult, error) {
	hashedToken := uc.TokenService.Hash(token)
	verification, err := uc.VerificationService.GetByToken(ctx, hashedToken)
	if err != nil {
		return nil, err
	}

	if verification == nil || uc.VerificationService.IsExpired(verification) {
		return nil, fmt.Errorf("invalid or expired token")
	}

	if verification.Type != models.TypeMagicLinkExchangeCode {
		return nil, fmt.Errorf("invalid token type")
	}

	user, err := uc.UserService.GetByID(ctx, *verification.UserID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	if err := uc.VerificationService.Delete(ctx, verification.ID); err != nil {
		return nil, err
	}

	sessionToken, err := uc.TokenService.Generate()
	if err != nil {
		return nil, err
	}
	hashedSessionToken := uc.TokenService.Hash(sessionToken)
	session, err := uc.SessionService.Create(
		ctx,
		user.ID,
		hashedSessionToken,
		ipAddress,
		userAgent,
		uc.GlobalConfig.Session.ExpiresIn,
	)
	if err != nil {
		return nil, err
	}

	return &types.MagicLinkExchangeResult{
		User:         user,
		Session:      session,
		SessionToken: sessionToken,
	}, nil
}
