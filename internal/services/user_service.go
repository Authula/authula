package services

import (
	"context"
	"errors"

	"github.com/GoBetterAuth/go-better-auth/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type userService struct {
	repo  repositories.UserRepository
	hooks *models.CoreDatabaseHooks
}

func NewUserService(repo repositories.UserRepository, hooks *models.CoreDatabaseHooks) services.UserService {
	return &userService{repo: repo, hooks: hooks}
}

func (s *userService) Create(ctx context.Context, name string, email string, emailVerified bool, image *string) (*models.User, error) {
	existing, _ := s.repo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, errors.New("email already in use")
	}

	user := &models.User{
		ID:            util.GenerateUUID(),
		Name:          name,
		Email:         email,
		EmailVerified: emailVerified,
		Image:         image,
	}

	if s.hooks != nil && s.hooks.Users != nil && s.hooks.Users.BeforeCreate != nil {
		if err := s.hooks.Users.BeforeCreate(user); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil && s.hooks.Users != nil && s.hooks.Users.AfterCreate != nil {
		if err := s.hooks.Users.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *userService) GetByID(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *userService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *userService) Update(ctx context.Context, user *models.User) (*models.User, error) {
	if s.hooks != nil && s.hooks.Users != nil && s.hooks.Users.BeforeUpdate != nil {
		if err := s.hooks.Users.BeforeUpdate(user); err != nil {
			return nil, err
		}
	}

	updatedUser, err := s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil && s.hooks.Users != nil && s.hooks.Users.AfterUpdate != nil {
		if err := s.hooks.Users.AfterUpdate(*updatedUser); err != nil {
			return nil, err
		}
	}

	return updatedUser, nil
}

func (s *userService) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	return s.repo.UpdateFields(ctx, id, fields)
}
