package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	repositories "github.com/Authula/authula/internal/repositories"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
)

type UsersService struct {
	userRepo   repositories.UserRepository
	authorizer rootservices.Authorizer
}

func NewUsersService(userRepo repositories.UserRepository, authorizer rootservices.Authorizer) *UsersService {
	return &UsersService{userRepo: userRepo, authorizer: authorizer}
}

func (s *UsersService) Create(ctx context.Context, actor *models.Actor, request types.CreateUserRequest) (*models.User, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersCreatePermission); err != nil {
		return nil, err
	}
	existing, err := s.userRepo.GetByEmail(ctx, request.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, internalerrors.ErrConflict
	}

	userToCreate := &models.User{
		ID:            util.GenerateUUID(),
		Name:          request.Name,
		Email:         request.Email,
		EmailVerified: *request.EmailVerified,
		Image:         request.Image,
		Metadata:      request.Metadata,
	}
	newUser, err := s.userRepo.Create(ctx, userToCreate)
	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *UsersService) GetAll(ctx context.Context, actor *models.Actor, cursor *string, limit int) (*types.UsersPage, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersListPermission); err != nil {
		return nil, err
	}
	users, nextCursor, err := s.userRepo.GetAll(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}

	return &types.UsersPage{Users: users, NextCursor: nextCursor}, nil
}

func (s *UsersService) GetByID(ctx context.Context, actor *models.Actor, userID string) (*models.User, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersReadPermission); err != nil {
		return nil, err
	}
	return s.userRepo.GetByID(ctx, userID)
}

func (s *UsersService) Update(ctx context.Context, actor *models.Actor, userID string, request types.UpdateUserRequest) (*models.User, error) {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersUpdatePermission); err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, internalerrors.ErrNotFound
	}

	if request.Name != nil {
		user.Name = *request.Name
	}
	if request.Email != nil {
		user.Email = *request.Email
	}
	if request.EmailVerified != nil {
		user.EmailVerified = *request.EmailVerified
	}
	if request.Image != nil {
		user.Image = request.Image
	}
	if len(request.Metadata) > 0 {
		user.Metadata = request.Metadata
	}

	updated, err := s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *UsersService) Delete(ctx context.Context, actor *models.Actor, userID string) error {
	if err := s.authorizer.AuthorizeScope(ctx, actor, adminconstants.UsersDeletePermission); err != nil {
		return err
	}
	existing, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return internalerrors.ErrNotFound
	}

	err = s.userRepo.Delete(ctx, userID)
	if err != nil {
		return err
	}
	return nil
}
