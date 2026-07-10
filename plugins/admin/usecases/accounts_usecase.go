package usecases

import (
	"context"
	"strings"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/services"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type AccountsUseCase struct {
	service    *services.AccountsService
	authorizer rootservices.Authorizer
}

func NewAccountsUseCase(service *services.AccountsService, authorizer rootservices.Authorizer) AccountsUseCase {
	return AccountsUseCase{service: service, authorizer: authorizer}
}

func (u AccountsUseCase) GetByID(ctx context.Context, actor *models.Actor, accountID string) (*models.Account, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsReadPermission); err != nil {
		return nil, err
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, coreerrors.ErrBadRequest
	}
	return u.service.GetByID(ctx, actor, accountID)
}

func (u AccountsUseCase) GetByUserID(ctx context.Context, actor *models.Actor, userID string) ([]models.Account, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsListPermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, adminconstants.ErrUserIDRequired
	}
	return u.service.GetByUserID(ctx, actor, userID)
}

func (u AccountsUseCase) Create(ctx context.Context, actor *models.Actor, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsCreatePermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	request.ProviderID = strings.TrimSpace(strings.ToLower(request.ProviderID))
	request.AccountID = strings.TrimSpace(request.AccountID)
	if request.Scope != nil {
		trimmed := strings.TrimSpace(*request.Scope)
		request.Scope = &trimmed
	}

	if userID == "" {
		return nil, adminconstants.ErrUserIDRequired
	}
	if request.ProviderID == "" || request.AccountID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.Create(ctx, actor, userID, request)
}

func (u AccountsUseCase) Update(ctx context.Context, actor *models.Actor, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsUpdatePermission); err != nil {
		return nil, err
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, coreerrors.ErrBadRequest
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
		return nil, coreerrors.ErrBadRequest
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

	return u.service.Update(ctx, actor, accountID, request)
}

func (u AccountsUseCase) Delete(ctx context.Context, actor *models.Actor, accountID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsDeletePermission); err != nil {
		return err
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return coreerrors.ErrBadRequest
	}
	return u.service.Delete(ctx, actor, accountID)
}
