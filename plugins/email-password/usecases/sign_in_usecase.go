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

type SignInUseCase struct {
	GlobalConfig                 *models.Config
	PluginConfig                 types.EmailPasswordPluginConfig
	Logger                       models.Logger
	UserService                  rootservices.UserService
	AccountService               rootservices.AccountService
	PasswordService              rootservices.PasswordService
	EventBus                     models.EventBus
	SendVerificationEmailUseCase *SendVerificationEmailUseCase
}

func (uc *SignInUseCase) SignIn(
	ctx context.Context,
	email string,
	password string,
	callbackURL *string,
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

	if uc.PluginConfig.RequireEmailVerification && !user.EmailVerified {
		if uc.PluginConfig.SendEmailOnSignIn {
			// Capture the current value of the pointer to avoid race conditions
			var cbURL string
			if callbackURL != nil {
				cbURL = *callbackURL
			}

			go func() {
				detachedCtx := context.WithoutCancel(ctx)
				taskCtx, cancel := context.WithTimeout(detachedCtx, 30*time.Second)
				defer cancel()

				err := uc.SendVerificationEmailUseCase.Send(taskCtx, user.Email, &cbURL)
				if err != nil {
					uc.Logger.Error("failed to send email", "err", err)
				}
			}()
		}

		return nil, constants.ErrEmailNotVerified
	}

	uc.publishSignInEvent(user)

	return &types.SignInResult{
		User: user,
	}, nil
}

func (uc *SignInUseCase) publishSignInEvent(user *models.User) {
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
