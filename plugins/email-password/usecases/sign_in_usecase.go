package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type SignInUseCase struct {
	GlobalConfig    *models.Config
	PluginConfig    types.EmailPasswordPluginConfig
	Logger          models.Logger
	UserService     rootservices.UserService
	AccountService  rootservices.AccountService
	SessionService  rootservices.SessionService
	TokenService    rootservices.TokenService
	PasswordService rootservices.PasswordService
	EventBus        models.EventBus
}

func (uc *SignInUseCase) SignIn(
	ctx context.Context,
	email string,
	password string,
	callbackURL *string,
	ipAddress *string,
	userAgent *string,
) (*types.SignInResult, error) {
	user, err := uc.UserService.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, constants.ErrInvalidCredentials
	}

	account, err := uc.AccountService.GetByUserIDAndProvider(ctx, user.ID, models.AuthProviderEmail.String())
	if err != nil {
		return nil, err
	}
	if account == nil || account.Password == nil || !uc.PasswordService.Verify(password, *account.Password) {
		return nil, constants.ErrInvalidCredentials
	}

	token, err := uc.TokenService.Generate()
	if err != nil {
		uc.Logger.Error("failed to generate session token", "error", err)
		return nil, err
	}

	hashedToken := uc.TokenService.Hash(token)

	newSession, err := uc.SessionService.Create(
		ctx,
		user.ID,
		hashedToken,
		ipAddress,
		userAgent,
		uc.GlobalConfig.Session.ExpiresIn,
	)
	if err != nil {
		uc.Logger.Error("failed to create session", "error", err)
		return nil, err
	}

	uc.publishSignedInEvent(user)

	return &types.SignInResult{
		User:         user,
		Session:      newSession,
		SessionToken: token,
	}, nil
}

func (uc *SignInUseCase) publishSignedInEvent(user *models.User) {
	userJson, err := json.Marshal(user)
	if err != nil {
		uc.Logger.Error(err.Error())
		return
	}

	util.PublishEventAsync(
		uc.EventBus,
		uc.Logger,
		models.Event{
			ID:        util.GenerateUUID(),
			Type:      constants.EventUserSignedIn,
			Payload:   userJson,
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}

func (uc *SignInUseCase) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	return uc.SessionService.GetByID(ctx, sessionID)
}

func (uc *SignInUseCase) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return uc.UserService.GetByID(ctx, userID)
}
