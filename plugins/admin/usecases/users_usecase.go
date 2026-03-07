package usecases

import (
	"context"
	"strings"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type UsersUseCase struct {
	service *services.UsersService
}

func NewUsersUseCase(service *services.UsersService) UsersUseCase {
	return UsersUseCase{service: service}
}

func (u UsersUseCase) Create(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	name := strings.TrimSpace(request.Name)
	email := strings.TrimSpace(strings.ToLower(request.Email))

	if name == "" {
		return nil, constants.ErrBadRequest
	}
	if email == "" {
		return nil, constants.ErrBadRequest
	}

	request.Name = name
	request.Email = email

	emailVerified := false
	if request.EmailVerified != nil {
		emailVerified = *request.EmailVerified
	}
	request.EmailVerified = &emailVerified

	return u.service.Create(ctx, request)
}

func (u UsersUseCase) GetAll(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	if limit <= 0 {
		limit = 10
	}

	if cursor != nil {
		trimmed := strings.TrimSpace(*cursor)
		cursor = &trimmed
	}

	return u.service.GetAll(ctx, cursor, limit)
}

func (u UsersUseCase) GetByID(ctx context.Context, userID string) (*models.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, constants.ErrUserIDRequired
	}

	return u.service.GetByID(ctx, userID)
}

func (u UsersUseCase) Update(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, constants.ErrUserIDRequired
	}
	if request.Name == nil && request.Email == nil && request.EmailVerified == nil && request.Image == nil && len(request.Metadata) == 0 {
		return nil, constants.ErrBadRequest
	}

	return u.service.Update(ctx, userID, request)
}

func (u UsersUseCase) Delete(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return constants.ErrBadRequest
	}

	return u.service.Delete(ctx, userID)
}
