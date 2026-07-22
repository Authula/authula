package usecases

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/services"
	"github.com/Authula/authula/plugins/email-password/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
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
	hooks           *services.ServiceHookExecutor
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
	hooks ...*services.ServiceHookExecutor,
) SignUpUseCase {
	var h *services.ServiceHookExecutor
	if len(hooks) > 0 {
		h = hooks[0]
	}
	return &signUpUseCase{GlobalConfig: globalConfig, PluginConfig: pluginConfig, Logger: logger, UserService: userService, AccountService: accountService, SessionService: sessionService, TokenService: tokenService, PasswordService: passwordService, EventBus: eventBus, hooks: h}
}

func (uc *signUpUseCase) SignUp(
	ctx context.Context,
	name string,
	email string,
	password string,
	image *string,
	metadata map[string]any,
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

	user := &models.User{
		Name:          name,
		Email:         email,
		EmailVerified: !uc.PluginConfig.RequireEmailVerification,
		Image:         image,
		Metadata:      metadata,
	}
	if len(user.Metadata) == 0 {
		user.Metadata = make(map[string]any)
	}

	if uc.hooks != nil {
		if err := uc.hooks.BeforeSignUp(ctx, user); err != nil {
			return nil, err
		}
	}

	createdUser, err := uc.UserService.Create(ctx, user.Name, user.Email, user.EmailVerified, user.Image, user.Metadata)
	if err != nil {
		return nil, err
	}

	hash, err := uc.PasswordService.Hash(password)
	if err != nil {
		return nil, err
	}

	_, err = uc.AccountService.Create(ctx, createdUser.ID, createdUser.Email, models.AuthProviderEmail.String(), &hash)
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
			createdUser.ID,
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

	if uc.hooks != nil {
		if err := uc.hooks.AfterSignUp(ctx, &types.SignUpResult{
			User:         createdUser,
			Session:      session,
			SessionToken: sessionToken,
		}); err != nil {
			return nil, err
		}
	}

	uc.publishSignedUpEvent(createdUser)

	return &types.SignUpResult{
		User:         createdUser,
		Session:      session,
		SessionToken: sessionToken,
	}, nil
}

func (uc *signUpUseCase) publishSignedUpEvent(user *models.User) {
	util.PublishEventAsync(
		uc.EventBus,
		uc.Logger,
		models.Event{
			ID:        util.GenerateUUID(),
			Type:      constants.EventUserSignedUp,
			Payload:   util.ToMap(user),
			Metadata:  nil,
			Timestamp: time.Now().UTC(),
		},
	)
}
