package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type userService struct {
	repo    repositories.UserRepository
	dbHooks *models.CoreDatabaseHooksConfig
}

func NewUserService(repo repositories.UserRepository, dbHooks *models.CoreDatabaseHooksConfig) services.UserService {
	return &userService{repo: repo, dbHooks: dbHooks}
}

func (s *userService) Create(ctx context.Context, name string, email string, emailVerified bool, image *string, metadata json.RawMessage) (*models.User, error) {
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
		Metadata:      metadata,
	}

	if s.dbHooks != nil && s.dbHooks.Users != nil && s.dbHooks.Users.BeforeCreate != nil {
		if err := s.dbHooks.Users.BeforeCreate(user); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	if s.dbHooks != nil && s.dbHooks.Users != nil && s.dbHooks.Users.AfterCreate != nil {
		if err := s.dbHooks.Users.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *userService) GetAll(ctx context.Context, cursor *string, limit int) ([]models.User, *string, error) {
	if limit <= 0 {
		limit = 10
	}

	return s.repo.GetAll(ctx, cursor, limit)
}

func (s *userService) GetByID(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *userService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *userService) Update(ctx context.Context, user *models.User) (*models.User, error) {
	if s.dbHooks != nil && s.dbHooks.Users != nil && s.dbHooks.Users.BeforeUpdate != nil {
		if err := s.dbHooks.Users.BeforeUpdate(user); err != nil {
			return nil, err
		}
	}

	updatedUser, err := s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	if s.dbHooks != nil && s.dbHooks.Users != nil && s.dbHooks.Users.AfterUpdate != nil {
		if err := s.dbHooks.Users.AfterUpdate(*updatedUser); err != nil {
			return nil, err
		}
	}

	return updatedUser, nil
}

func (s *userService) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	return s.repo.UpdateFields(ctx, id, fields)
}

func (s *userService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
