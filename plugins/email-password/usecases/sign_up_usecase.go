package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
	rootservices "github.com/Authula/authula/services"
)

type signUpUseCase struct {
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

func NewSignUpUseCase(
	globalConfig *models.Config,
	pluginConfig types.EmailPasswordPluginConfig,
	logger models.Logger,
	userService rootservices.UserService,
	accountService rootservices.AccountService,
	sessionService rootservices.SessionService,
	tokenService rootservices.TokenService,
	passwordService rootservices.PasswordService,
	eventBus models.EventBus,
) SignUpUseCase {
	return &signUpUseCase{GlobalConfig: globalConfig, PluginConfig: pluginConfig, Logger: logger, UserService: userService, AccountService: accountService, SessionService: sessionService, TokenService: tokenService, PasswordService: passwordService, EventBus: eventBus}
}

func (uc *signUpUseCase) SignUp(
	ctx context.Context,
	name string,
	email string,
	password string,
	image *string,
	metadata json.RawMessage,
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

	if existing, err := uc.UserService.GetByEmail(ctx, email); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, constants.ErrEmailAlreadyExists
	}

	hash, err := uc.PasswordService.Hash(password)
	if err != nil {
		return nil, err
	}

	user, err := uc.UserService.Create(ctx, name, email, !uc.PluginConfig.RequireEmailVerification, image, metadata)
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

func (uc *signUpUseCase) publishSignedUpEvent(user *models.User) {
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
