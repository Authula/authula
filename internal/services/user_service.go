package services

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type UserServiceImpl struct {
	config *models.Config
	db     *gorm.DB
}

func NewUserServiceImpl(config *models.Config, db *gorm.DB) *UserServiceImpl {
	return &UserServiceImpl{config: config, db: db}
}

// CreateUser creates a new user in the database.
func (s *UserServiceImpl) CreateUser(user *models.User) error {
	user.ID = uuid.NewString()
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	if s.config.DatabaseHooks.Users != nil && s.config.DatabaseHooks.Users.BeforeCreate != nil {
		if err := s.config.DatabaseHooks.Users.BeforeCreate(user); err != nil {
			return err
		}
	}

	if err := s.db.Create(user).Error; err != nil {
		return err
	}

	if s.config.DatabaseHooks.Users != nil && s.config.DatabaseHooks.Users.AfterCreate != nil {
		go func() {
			if err := s.config.DatabaseHooks.Users.AfterCreate(*user); err != nil {
				slog.Error("user after create hook failed", "error", err.Error())
			}
		}()
	}

	return nil
}

// GetUserByID retrieves a user by their ID.
func (s *UserServiceImpl) GetUserByID(id string) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by their email.
func (s *UserServiceImpl) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user in the database.
func (s *UserServiceImpl) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now().UTC()

	if s.config.DatabaseHooks.Users != nil && s.config.DatabaseHooks.Users.BeforeUpdate != nil {
		if err := s.config.DatabaseHooks.Users.BeforeUpdate(user); err != nil {
			return err
		}
	}

	if err := s.db.Save(user).Error; err != nil {
		return err
	}

	if s.config.DatabaseHooks.Users != nil && s.config.DatabaseHooks.Users.AfterUpdate != nil {
		go func() {
			if err := s.config.DatabaseHooks.Users.AfterUpdate(*user); err != nil {
				slog.Error("user after update hook failed", "error", err.Error())
			}
		}()
	}

	return nil
}
