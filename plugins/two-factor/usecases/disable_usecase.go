package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/two-factor/repository"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type disableUseCase struct {
	AccountService  rootservices.AccountService
	PasswordService rootservices.PasswordService
	Repo            *repository.TwoFactorRepository
	EventBus        models.EventBus
	Logger          models.Logger
}

func NewDisableUseCase(
	accountService rootservices.AccountService,
	passwordService rootservices.PasswordService,
	repo *repository.TwoFactorRepository,
	eventBus models.EventBus,
	logger models.Logger,
) DisableUseCase {
	return &disableUseCase{
		AccountService:  accountService,
		PasswordService: passwordService,
		Repo:            repo,
		EventBus:        eventBus,
		Logger:          logger,
	}
}

func (uc *disableUseCase) Disable(ctx context.Context, userID, password string) error {
	// Verify password
	if err := verifyPassword(ctx, uc.AccountService, uc.PasswordService, userID, password); err != nil {
		return err
	}

	// Check that 2FA is enabled
	existing, err := uc.Repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return constants.ErrTwoFactorNotEnabled
	}

	// Delete two_factor record
	if err := uc.Repo.DeleteByUserID(ctx, userID); err != nil {
		return err
	}

	// Set user.two_factor_enabled = false
	if err := uc.Repo.SetUserTwoFactorEnabled(ctx, userID, false); err != nil {
		return err
	}

	// Delete trusted devices
	if err := uc.Repo.DeleteTrustedDevicesByUserID(ctx, userID); err != nil {
		return err
	}

	// Publish disabled event
	payload, err := json.Marshal(map[string]string{"userID": userID})
	if err != nil {
		uc.Logger.Error(err.Error())
	} else {
		util.PublishEventAsync(
			uc.EventBus,
			uc.Logger,
			models.Event{
				ID:        util.GenerateUUID(),
				Type:      constants.EventTwoFactorDisabled,
				Payload:   payload,
				Metadata:  nil,
				Timestamp: time.Now().UTC(),
			},
		)
	}

	return nil
}
