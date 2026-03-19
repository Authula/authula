package usecases

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/constants"
)

type DisableUseCase struct {
	Logger   models.Logger
	EventBus models.EventBus
	TOTPRepo TOTPRepository
}

func NewDisableUseCase(
	logger models.Logger,
	eventBus models.EventBus,
	totpRepo TOTPRepository,
) *DisableUseCase {
	return &DisableUseCase{
		Logger:   logger,
		EventBus: eventBus,
		TOTPRepo: totpRepo,
	}
}

func (uc *DisableUseCase) Disable(ctx context.Context, userID string) error {
	existing, err := uc.TOTPRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return constants.ErrTOTPNotEnabled
	}

	if err := uc.TOTPRepo.DeleteByUserID(ctx, userID); err != nil {
		return err
	}

	if err := uc.TOTPRepo.DeleteTrustedDevicesByUserID(ctx, userID); err != nil {
		return err
	}

	publishEvent(uc.EventBus, uc.Logger, constants.EventTOTPDisabled, userID)

	return nil
}
