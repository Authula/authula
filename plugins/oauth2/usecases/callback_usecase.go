package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
	rootservices "github.com/Authula/authula/services"
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

	user, err := uc.resolveUser(ctx, req.ProviderID, userInfo)
	if err != nil {
		return nil, err
	}

	var accessTokenExpiry *time.Time
	if !token.Expiry.IsZero() {
		accessTokenExpiry = &token.Expiry
	}

	// CreateOAuth2 upserts by (provider, account ID) and encrypts the tokens.
	if _, err := uc.AccountService.CreateOAuth2(
		ctx,
		user.ID,
		userInfo.ProviderAccountID,
		req.ProviderID,
		token.AccessToken,
		&token.RefreshToken,
		accessTokenExpiry,
		nil,
		nil,
	); err != nil {
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

// resolveUser finds the user for a provider identity. A known provider account always wins over email,
// because emails are mutable. Linking to an existing user by email is only allowed when the provider
// verified that email; otherwise anyone controlling the provider profile could take over the account.
func (uc *CallbackUseCase) resolveUser(ctx context.Context, providerID string, userInfo *types.UserInfo) (*models.User, error) {
	if userInfo.ProviderAccountID == "" {
		return nil, fmt.Errorf("provider did not return an account id")
	}
	if userInfo.Email == "" {
		return nil, fmt.Errorf("provider did not return an email address")
	}

	account, err := uc.AccountService.GetByProviderAndAccountID(ctx, providerID, userInfo.ProviderAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}
	if account != nil {
		user, err := uc.UserService.GetByID(ctx, account.UserID)
		if err != nil {
			return nil, fmt.Errorf("database error checking user: %w", err)
		}
		if user == nil {
			return nil, fmt.Errorf("linked user not found")
		}
		return user, nil
	}

	user, err := uc.UserService.GetByEmail(ctx, userInfo.Email)
	if err != nil {
		return nil, fmt.Errorf("database error checking user: %w", err)
	}
	if user != nil {
		if !userInfo.EmailVerified {
			return nil, fmt.Errorf("email is not verified by %s; sign in with your existing method first", providerID)
		}
		return user, nil
	}

	user, err = uc.UserService.Create(ctx, userInfo.Name, userInfo.Email, userInfo.EmailVerified, &userInfo.Picture, nil)
	if err != nil {
		uc.Logger.Error(fmt.Sprintf("Failed to create user: %v", err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (uc *CallbackUseCase) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	return uc.SessionService.GetByID(ctx, sessionID)
}

func (uc *CallbackUseCase) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return uc.UserService.GetByID(ctx, userID)
}
