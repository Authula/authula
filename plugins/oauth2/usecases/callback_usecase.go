package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/services"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type CallbackUseCase struct {
	GlobalConfig     *models.Config
	ProviderRegistry *services.ProviderRegistry
	Logger           models.Logger
	HMACKey          []byte
	UserService      rootservices.UserService
	AccountService   rootservices.AccountService
	SessionService   rootservices.SessionService
	TokenService     rootservices.TokenService
}

func NewCallbackUseCase(
	globalConfig *models.Config,
	registry *services.ProviderRegistry,
	logger models.Logger,
	hmacKey []byte,
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	sessionService rootservices.SessionService,
	tokenService rootservices.TokenService,
) *CallbackUseCase {
	return &CallbackUseCase{
		GlobalConfig:     globalConfig,
		ProviderRegistry: registry,
		Logger:           logger,
		HMACKey:          hmacKey,
		UserService:      userService,
		AccountService:   accountService,
		SessionService:   sessionService,
		TokenService:     tokenService,
	}
}

func (uc *CallbackUseCase) Callback(ctx context.Context, req *types.CallbackRequest, ipAddress *string, userAgent *string) (*types.CallbackResult, error) {
	if req.Error != "" {
		return nil, fmt.Errorf("oauth provider error: %s", req.Error)
	}

	oauthProvider, exists := uc.ProviderRegistry.Get(req.ProviderID)
	if !exists {
		return nil, fmt.Errorf("provider %s not found", req.ProviderID)
	}

	token, err := oauthProvider.Exchange(ctx, req.Code)
	if err != nil {
		uc.Logger.Error(fmt.Sprintf("Failed to exchange code: %v", err))
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	userInfo, err := oauthProvider.GetUserInfo(ctx, token)
	if err != nil {
		uc.Logger.Error(fmt.Sprintf("Failed to get user info: %v", err))
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := uc.UserService.GetByEmail(ctx, userInfo.Email)
	if err != nil {
		return nil, fmt.Errorf("database error checking user: %w", err)
	}

	if user == nil {
		user, err = uc.UserService.Create(ctx, userInfo.Name, userInfo.Email, true, &userInfo.Picture)
		if err != nil {
			uc.Logger.Error(fmt.Sprintf("Failed to create user: %v", err))
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	existingAccount, err := uc.AccountService.GetByProviderAndAccountID(ctx, req.ProviderID, userInfo.ProviderAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}

	var accessTokenExpiry *time.Time
	if !token.Expiry.IsZero() {
		accessTokenExpiry = &token.Expiry
	}

	if existingAccount == nil {
		_, err = uc.AccountService.CreateOAuth2(
			ctx,
			user.ID,
			userInfo.ProviderAccountID,
			req.ProviderID,
			token.AccessToken,
			&token.RefreshToken,
			accessTokenExpiry,
			nil,
			nil,
		)
	} else {
		err = uc.AccountService.UpdateFields(ctx, existingAccount.ID, map[string]any{
			"access_token":            token.AccessToken,
			"refresh_token":           token.RefreshToken,
			"access_token_expires_at": accessTokenExpiry,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to handle account linking: %w", err)
	}

	sessionToken, err := uc.TokenService.Generate()
	if err != nil {
		uc.Logger.Error("failed to generate session token", "error", err)
		return nil, err
	}

	hashedSessionToken := uc.TokenService.Hash(sessionToken)

	newSession, err := uc.SessionService.Create(
		ctx,
		user.ID,
		hashedSessionToken,
		ipAddress,
		userAgent,
		uc.GlobalConfig.Session.ExpiresIn,
	)
	if err != nil {
		uc.Logger.Error("failed to create session", "error", err)
		return nil, err
	}

	return &types.CallbackResult{
		User:         user,
		Session:      newSession,
		SessionToken: sessionToken,
	}, nil
}

func (uc *CallbackUseCase) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	return uc.SessionService.GetByID(ctx, sessionID)
}

func (uc *CallbackUseCase) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return uc.UserService.GetByID(ctx, userID)
}
