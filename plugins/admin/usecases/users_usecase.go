package usecases

import (
	"context"

	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/services"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type UsersUseCase struct {
	service    *services.UsersService
	authorizer rootservices.Authorizer
}

func NewUsersUseCase(service *services.UsersService, authorizer rootservices.Authorizer) UsersUseCase {
	return UsersUseCase{service: service, authorizer: authorizer}
}

func (u UsersUseCase) Create(ctx context.Context, actor *models.Actor, request types.CreateUserRequest) (*models.User, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersCreatePermission); err != nil {
		return nil, err
	}
	return u.service.Create(ctx, actor, request)
}

func (u UsersUseCase) GetAll(ctx context.Context, actor *models.Actor, cursor *string, limit int) (*types.UsersPage, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersListPermission); err != nil {
		return nil, err
	}
	return u.service.GetAll(ctx, actor, cursor, limit)
}

func (u UsersUseCase) GetByID(ctx context.Context, actor *models.Actor, userID string) (*models.User, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersReadPermission); err != nil {
		return nil, err
	}
	return u.service.GetByID(ctx, actor, userID)
}

func (u UsersUseCase) Update(ctx context.Context, actor *models.Actor, userID string, request types.UpdateUserRequest) (*models.User, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersUpdatePermission); err != nil {
		return nil, err
	}
	return u.service.Update(ctx, actor, userID, request)
}

func (u UsersUseCase) Delete(ctx context.Context, actor *models.Actor, userID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersDeletePermission); err != nil {
		return err
	}
	return u.service.Delete(ctx, actor, userID)
}
