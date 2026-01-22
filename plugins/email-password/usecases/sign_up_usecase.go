package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/email-password/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type SignUpUseCase struct {
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

func (uc *SignUpUseCase) SignUp(
	ctx context.Context,
	name string,
	email string,
	password string,
	image *string,
	callbackURL *string,
	ipAddress *string,
	userAgent *string,
) (*types.SignUpResult, error) {
	if uc.PluginConfig.DisableSignUp {
		return nil, constants.ErrSignUpDisabled
	}

	if len(password) < uc.PluginConfig.MinPasswordLength || len(password) > uc.PluginConfig.MaxPasswordLength {
		return nil, constants.ErrInvalidPasswordLength
	}

	if existing, _ := uc.UserService.GetByEmail(ctx, email); existing != nil {
		return nil, constants.ErrEmailAlreadyExists
	}

	hash, err := uc.PasswordService.Hash(password)
	if err != nil {
		return nil, err
	}

	user, err := uc.UserService.Create(ctx, name, email, !uc.PluginConfig.RequireEmailVerification, image)
	if err != nil {
		return nil, err
	}

	_, err = uc.AccountService.Create(ctx, user.ID, user.Email, models.AuthProviderEmail.String(), &hash)
	if err != nil {
		return nil, err
	}

	var session *models.Session
	sessionToken := ""

	if uc.PluginConfig.AutoSignIn {
		token, err := uc.TokenService.Generate()
		if err != nil {
			uc.Logger.Error("failed to generate session token", "error", err)
			return nil, err
		}
		sessionToken = token

		hashedToken := uc.TokenService.Hash(token)

		session, err = uc.SessionService.Create(
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
	}

	uc.publishSignedUpEvent(user)

	return &types.SignUpResult{
		User:         user,
		Session:      session,
		SessionToken: sessionToken,
	}, nil
}

func (uc *SignUpUseCase) publishSignedUpEvent(user *models.User) {
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
			Type:      constants.EventUserSignedUp,
			Payload:   userJson,
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}
