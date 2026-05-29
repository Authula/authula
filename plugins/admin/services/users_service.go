package services

import (
	"context"

	internalerrors "github.com/Authula/authula/internal/errors"
	repositories "github.com/Authula/authula/internal/repositories"
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/types"
)

type UsersService struct {
	userRepo repositories.UserRepository
}

func NewUsersService(userRepo repositories.UserRepository) *UsersService {
	return &UsersService{userRepo: userRepo}
}

func (s *UsersService) Create(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
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

func (s *UsersService) GetAll(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	users, nextCursor, err := s.userRepo.GetAll(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}

	return &types.UsersPage{Users: users, NextCursor: nextCursor}, nil
}

func (s *UsersService) GetByID(ctx context.Context, userID string) (*models.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *UsersService) Update(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
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

func (s *UsersService) Delete(ctx context.Context, userID string) error {
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
