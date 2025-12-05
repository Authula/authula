package auth

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type AccountService struct {
	config *domain.Config
	db     *gorm.DB
}

func NewAccountService(config *domain.Config, db *gorm.DB) *AccountService {
	return &AccountService{config: config, db: db}
}

// CreateAccount creates a new account in the database.
func (s *AccountService) CreateAccount(a *domain.Account) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = time.Now().UTC()

	if s.config.DatabaseHooks.Accounts != nil && s.config.DatabaseHooks.Accounts.BeforeCreate != nil {
		if err := s.config.DatabaseHooks.Accounts.BeforeCreate(a); err != nil {
			return err
		}
	}

	if err := s.db.Create(a).Error; err != nil {
		return err
	}

	if s.config.DatabaseHooks.Accounts != nil && s.config.DatabaseHooks.Accounts.AfterCreate != nil {
		go func() {
			if err := s.config.DatabaseHooks.Accounts.AfterCreate(*a); err != nil {
				slog.Error("account after create hook failed", "error", err.Error())
			}
		}()
	}

	return nil
}

// GetAccountByUserID retrieves an account by the associated user ID.
func (s *AccountService) GetAccountByUserID(userID string) (*domain.Account, error) {
	var account domain.Account
	if err := s.db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// GetAccountByProviderAndAccountID retrieves an account by provider and provider's account ID.
func (s *AccountService) GetAccountByProviderAndAccountID(provider domain.ProviderType, accountID string) (*domain.Account, error) {
	var account domain.Account
	if err := s.db.Where("provider_id = ? AND account_id = ?", provider, accountID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// UpdateAccount updates an existing account in the database.
func (s *AccountService) UpdateAccount(account *domain.Account) error {
	account.UpdatedAt = time.Now().UTC()

	if s.config.DatabaseHooks.Accounts != nil && s.config.DatabaseHooks.Accounts.BeforeUpdate != nil {
		if err := s.config.DatabaseHooks.Accounts.BeforeUpdate(account); err != nil {
			return err
		}
	}

	result := s.db.Model(&domain.Account{}).Where("id = ?", account.ID).Updates(account)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if s.config.DatabaseHooks.Accounts != nil && s.config.DatabaseHooks.Accounts.AfterUpdate != nil {
		go func() {
			if err := s.config.DatabaseHooks.Accounts.AfterUpdate(*account); err != nil {
				slog.Error("account after update hook failed", "error", err.Error())
			}
		}()
	}

	return nil
}
