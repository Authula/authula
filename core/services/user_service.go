package services

import (
	"context"
	"errors"

	"github.com/Authula/authula/core/repositories"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type userService struct {
	repo         repositories.UserRepository
	serviceHooks *models.CoreServiceHooksConfig
	logger       models.Logger
}

func NewUserService(repo repositories.UserRepository, serviceHooks *models.CoreServiceHooksConfig, logger models.Logger) services.UserService {
	return &userService{repo: repo, serviceHooks: serviceHooks, logger: logger}
}

func (s *userService) Create(ctx context.Context, name string, email string, emailVerified bool, image *string, metadata map[string]any) (*models.User, error) {
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

	if s.serviceHooks != nil && s.serviceHooks.Users != nil {
		if err := runHooks(s.serviceHooks.Users.BeforeCreateHooks(), user); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Users != nil {
		runAfterHooks(s.logger, "after create user", s.serviceHooks.Users.AfterCreateHooks(), created)
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
	if s.serviceHooks != nil && s.serviceHooks.Users != nil {
		if err := runHooks(s.serviceHooks.Users.BeforeUpdateHooks(), user); err != nil {
			return nil, err
		}
	}

	updatedUser, err := s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Users != nil {
		runAfterHooks(s.logger, "after update user", s.serviceHooks.Users.AfterUpdateHooks(), updatedUser)
	}

	return updatedUser, nil
}

func (s *userService) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	return s.repo.UpdateFields(ctx, id, fields)
}

func (s *userService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
