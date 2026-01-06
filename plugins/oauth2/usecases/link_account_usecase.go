package usecases

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/services"
	rootservices "github.com/GoBetterAuth/go-better-auth/services"
)

type LinkAccountUseCase struct {
	ProviderRegistry *services.ProviderRegistry
	Logger           models.Logger
	UserService      rootservices.UserService
	AccountService   rootservices.AccountService
}

func NewLinkAccountUseCase(
	registry *services.ProviderRegistry,
	logger models.Logger,
	userService rootservices.UserService,
	accountService rootservices.AccountService,
) *LinkAccountUseCase {
	return &LinkAccountUseCase{
		ProviderRegistry: registry,
		Logger:           logger,
		UserService:      userService,
		AccountService:   accountService,
	}
}

func (uc *LinkAccountUseCase) LinkAccount(ctx context.Context, userID, providerID, providerAccountID string) (*LinkAccountResult, error) {
	_, exists := uc.ProviderRegistry.Get(providerID)
	if !exists {
		return &LinkAccountResult{
			Success: false,
			Error:   fmt.Sprintf("provider %s not found", providerID),
		}, fmt.Errorf("provider %s not found", providerID)
	}

	account := &models.Account{
		UserID:     userID,
		ProviderID: providerID,
		AccountID:  providerAccountID,
	}

	if _, err := uc.AccountService.Update(ctx, account); err != nil {
		return &LinkAccountResult{
			Success: false,
			Error:   fmt.Sprintf("failed to link account: %v", err),
		}, fmt.Errorf("failed to link account: %w", err)
	}

	return &LinkAccountResult{
		Success:    true,
		ProviderID: providerID,
		AccountID:  providerAccountID,
	}, nil
}
