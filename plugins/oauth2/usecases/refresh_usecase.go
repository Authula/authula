package usecases

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/services"
)

type RefreshUseCase struct {
	ProviderRegistry *services.ProviderRegistry
	Logger           models.Logger
}

func NewRefreshUseCase(
	registry *services.ProviderRegistry,
	logger models.Logger,
) *RefreshUseCase {
	return &RefreshUseCase{
		ProviderRegistry: registry,
		Logger:           logger,
	}
}

func (uc *RefreshUseCase) Refresh(ctx context.Context, userID, providerID string) (*RefreshResult, error) {
	oauthProvider, exists := uc.ProviderRegistry.Get(providerID)
	if !exists {
		return &RefreshResult{
			Success: false,
			Error:   fmt.Sprintf("provider %s not found", providerID),
		}, fmt.Errorf("provider %s not found", providerID)
	}

	// TODO: Get stored token from account store and refresh it
	// This will be implemented once we have the full storage interface

	_ = oauthProvider

	return &RefreshResult{
		Success: false,
		Error:   "not implemented",
	}, nil
}
