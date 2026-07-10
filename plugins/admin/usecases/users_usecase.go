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
	name := strings.TrimSpace(request.Name)
	email := strings.TrimSpace(strings.ToLower(request.Email))

	if name == "" {
		return nil, coreerrors.ErrBadRequest
	}
	if email == "" {
		return nil, coreerrors.ErrBadRequest
	}

	request.Name = name
	request.Email = email

	emailVerified := false
	if request.EmailVerified != nil {
		emailVerified = *request.EmailVerified
	}
	request.EmailVerified = &emailVerified

	return u.service.Create(ctx, actor, request)
}

func (u UsersUseCase) GetAll(ctx context.Context, actor *models.Actor, cursor *string, limit int) (*types.UsersPage, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersListPermission); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	if cursor != nil {
		trimmed := strings.TrimSpace(*cursor)
		cursor = &trimmed
	}

	return u.service.GetAll(ctx, actor, cursor, limit)
}

func (u UsersUseCase) GetByID(ctx context.Context, actor *models.Actor, userID string) (*models.User, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersReadPermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, adminconstants.ErrUserIDRequired
	}

	return u.service.GetByID(ctx, actor, userID)
}

func (u UsersUseCase) Update(ctx context.Context, actor *models.Actor, userID string, request types.UpdateUserRequest) (*models.User, error) {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersUpdatePermission); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, adminconstants.ErrUserIDRequired
	}
	if request.Name == nil && request.Email == nil && request.EmailVerified == nil && request.Image == nil && len(request.Metadata) == 0 {
		return nil, coreerrors.ErrBadRequest
	}

	return u.service.Update(ctx, actor, userID, request)
}

func (u UsersUseCase) Delete(ctx context.Context, actor *models.Actor, userID string) error {
	if err := u.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersDeletePermission); err != nil {
		return err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return coreerrors.ErrBadRequest
	}

	return u.service.Delete(ctx, actor, userID)
}
