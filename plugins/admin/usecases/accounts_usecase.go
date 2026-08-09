package usecases

import (
	"context"

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
	return u.service.GetByID(ctx, actor, accountID)
}

func (u AccountsUseCase) GetByUserID(ctx context.Context, actor *models.Actor, userID string) ([]models.Account, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsListPermission); err != nil {
		return nil, err
	}
	return u.service.GetByUserID(ctx, actor, userID)
}

func (u AccountsUseCase) Create(ctx context.Context, actor *models.Actor, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsCreatePermission); err != nil {
		return nil, err
	}
	return u.service.Create(ctx, actor, userID, request)
}

func (u AccountsUseCase) Update(ctx context.Context, actor *models.Actor, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsUpdatePermission); err != nil {
		return nil, err
	}
	return u.service.Update(ctx, actor, accountID, request)
}

func (u AccountsUseCase) Delete(ctx context.Context, actor *models.Actor, accountID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.AccountsDeletePermission); err != nil {
		return err
	}
	return u.service.Delete(ctx, actor, accountID)
}
