package usecases

import (
	"context"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type AccountsUseCase struct {
	service *services.AccountsService
}

func NewAccountsUseCase(service *services.AccountsService) AccountsUseCase {
	return AccountsUseCase{service: service}
}

func (u AccountsUseCase) GetByID(ctx context.Context, accountID string) (*models.Account, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, constants.ErrBadRequest
	}
	return u.service.GetByID(ctx, accountID)
}

func (u AccountsUseCase) GetByUserID(ctx context.Context, userID string) ([]models.Account, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, constants.ErrUserIDRequired
	}
	return u.service.GetByUserID(ctx, userID)
}

func (u AccountsUseCase) Create(ctx context.Context, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	userID = strings.TrimSpace(userID)
	request.ProviderID = strings.TrimSpace(strings.ToLower(request.ProviderID))
	request.AccountID = strings.TrimSpace(request.AccountID)
	if request.Scope != nil {
		trimmed := strings.TrimSpace(*request.Scope)
		request.Scope = &trimmed
	}

	if userID == "" {
		return nil, constants.ErrUserIDRequired
	}
	if request.ProviderID == "" || request.AccountID == "" {
		return nil, constants.ErrBadRequest
	}

	return u.service.Create(ctx, userID, request)
}

func (u AccountsUseCase) Update(ctx context.Context, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, constants.ErrBadRequest
	}

	if request.ProviderID == nil &&
		request.AccountID == nil &&
		request.AccessToken == nil &&
		request.RefreshToken == nil &&
		request.IDToken == nil &&
		request.AccessTokenExpiresAt == nil &&
		request.RefreshTokenExpiresAt == nil &&
		request.Scope == nil &&
		request.Password == nil {
		return nil, constants.ErrBadRequest
	}

	if request.ProviderID != nil {
		trimmed := strings.TrimSpace(strings.ToLower(*request.ProviderID))
		request.ProviderID = &trimmed
	}
	if request.AccountID != nil {
		trimmed := strings.TrimSpace(*request.AccountID)
		request.AccountID = &trimmed
	}
	if request.Scope != nil {
		trimmed := strings.TrimSpace(*request.Scope)
		request.Scope = &trimmed
	}

	return u.service.Update(ctx, accountID, request)
}

func (u AccountsUseCase) Delete(ctx context.Context, accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return constants.ErrBadRequest
	}
	return u.service.Delete(ctx, accountID)
}
