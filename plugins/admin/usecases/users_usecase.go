package usecases

import (
	"context"
	"errors"
	"strings"

	repositories "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type usersUseCase struct {
	userRepo repositories.UserRepository
}

func NewUsersUseCase(userRepo repositories.UserRepository) UsersUseCase {
	return &usersUseCase{userRepo: userRepo}
}

func (u *usersUseCase) Create(ctx context.Context, request types.CreateUserRequest) (*models.User, error) {
	name := strings.TrimSpace(request.Name)
	email := strings.TrimSpace(strings.ToLower(request.Email))

	if name == "" {
		return nil, errors.New("name is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}

	existing, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user already exists")
	}

	emailVerified := false
	if request.EmailVerified != nil {
		emailVerified = *request.EmailVerified
	}

	userToCreate := &models.User{
		Name:          name,
		Email:         email,
		EmailVerified: emailVerified,
		Image:         request.Image,
		Metadata:      request.Metadata,
	}
	newUser, err := u.userRepo.Create(ctx, userToCreate)
	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func (u *usersUseCase) GetAll(ctx context.Context, cursor *string, limit int) (*types.UsersPage, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if cursor != nil {
		trimmed := strings.TrimSpace(*cursor)
		cursor = &trimmed
	}

	users, nextCursor, err := u.userRepo.GetAll(ctx, cursor, limit)
	if err != nil {
		return nil, err
	}

	return &types.UsersPage{Users: users, NextCursor: nextCursor}, nil
}

func (u *usersUseCase) GetByID(ctx context.Context, userID string) (*models.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}

	return u.userRepo.GetByID(ctx, userID)
}

func (u *usersUseCase) Update(ctx context.Context, userID string, request types.UpdateUserRequest) (*models.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if request.Name == nil && request.Email == nil && request.EmailVerified == nil && request.Image == nil && len(request.Metadata) == 0 {
		return nil, errors.New("at least one field is required for update")
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
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

	updated, err := u.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (u *usersUseCase) Delete(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("user_id is required")
	}

	existing, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("user not found")
	}

	err = u.userRepo.Delete(ctx, userID)
	if err != nil {
		return err
	}
	return nil
}
