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
	GlobalConfig                 *models.Config
	PluginConfig                 types.EmailPasswordPluginConfig
	Logger                       models.Logger
	UserService                  rootservices.UserService
	AccountService               rootservices.AccountService
	PasswordService              rootservices.PasswordService
	SendVerificationEmailUseCase *SendVerificationEmailUseCase
	EventBus                     models.EventBus
}

func (uc *SignUpUseCase) SignUp(
	ctx context.Context,
	name string,
	email string,
	password string,
	image *string,
	callbackURL *string,
) (*types.SignUpResult, error) {
	if uc.PluginConfig.DisableSignUp {
		return nil, constants.ErrSignUpDisabled
	}

	if len(password) < uc.PluginConfig.MinPasswordLength || len(password) > uc.PluginConfig.MaxPasswordLength {
		return nil, constants.ErrPasswordLengthInvalid
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

	if uc.PluginConfig.RequireEmailVerification {
		// Capture the current value of the pointer to avoid race conditions
		var cbURL string
		if callbackURL != nil {
			cbURL = *callbackURL
		}

		go func() {
			detachedCtx := context.WithoutCancel(ctx)
			taskCtx, cancel := context.WithTimeout(detachedCtx, 30*time.Second)
			defer cancel()

			err := uc.SendVerificationEmailUseCase.Send(taskCtx, email, &cbURL)
			if err != nil {
				uc.Logger.Error("failed to send email", "err", err)
			}
		}()
	}

	uc.publishSignUpEvent(user)

	return &types.SignUpResult{
		User: user,
	}, nil
}

func (uc *SignUpUseCase) publishSignUpEvent(user *models.User) {
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
