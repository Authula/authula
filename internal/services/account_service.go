package services

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type accountService struct {
	config      *models.Config
	accountRepo repositories.AccountRepository
	tokenRepo   repositories.TokenRepository
	dbHooks     *models.CoreDatabaseHooksConfig
}

func NewAccountService(
	config *models.Config,
	accountRepo repositories.AccountRepository,
	tokenRepo repositories.TokenRepository,
	dbHooks *models.CoreDatabaseHooksConfig,
) services.AccountService {
	return &accountService{config: config, accountRepo: accountRepo, tokenRepo: tokenRepo, dbHooks: dbHooks}
}
func (s *accountService) Create(ctx context.Context, userID string, accountID string, providerID string, password *string) (*models.Account, error) {
	account := &models.Account{
		ID:         util.GenerateUUID(),
		UserID:     userID,
		AccountID:  accountID,
		ProviderID: providerID,
		Password:   password,
	}

	if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.BeforeCreate != nil {
		if err := s.dbHooks.Accounts.BeforeCreate(account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.AfterCreate != nil {
		if err := s.dbHooks.Accounts.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *accountService) CreateOAuth2(ctx context.Context, userID string, providerAccountID string, provider string, accessToken string, refreshToken *string, accessTokenExpiresAt *time.Time, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error) {
	encryptedAccessToken, err := s.tokenRepo.Encrypt(accessToken)
	if err != nil {
		return nil, err
	}

	var encryptedRefreshToken *string
	if refreshToken != nil {
		encrypted, err := s.tokenRepo.Encrypt(*refreshToken)
		if err != nil {
			return nil, err
		}
		encryptedRefreshToken = &encrypted
	}

	existing, err := s.accountRepo.GetByProviderAndAccountID(ctx, provider, providerAccountID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update existing account
		existing.AccessToken = &encryptedAccessToken
		existing.RefreshToken = encryptedRefreshToken
		existing.AccessTokenExpiresAt = accessTokenExpiresAt
		existing.RefreshTokenExpiresAt = refreshTokenExpiresAt
		existing.Scope = scope

		if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.BeforeUpdate != nil {
			if err := s.dbHooks.Accounts.BeforeUpdate(existing); err != nil {
				return nil, err
			}
		}

		updated, err := s.accountRepo.Update(ctx, existing)
		if err != nil {
			return nil, err
		}

		if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.AfterUpdate != nil {
			if err := s.dbHooks.Accounts.AfterUpdate(*updated); err != nil {
				return nil, err
			}
		}

		return updated, nil
	}

	account := &models.Account{
		ID:                    util.GenerateUUID(),
		UserID:                userID,
		AccountID:             providerAccountID,
		ProviderID:            provider,
		AccessToken:           &encryptedAccessToken,
		RefreshToken:          encryptedRefreshToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
		Password:              nil,
		Scope:                 scope,
	}

	if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.BeforeCreate != nil {
		if err := s.dbHooks.Accounts.BeforeCreate(account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.AfterCreate != nil {
		if err := s.dbHooks.Accounts.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *accountService) GetByUserID(ctx context.Context, userID string) (*models.Account, error) {
	return s.accountRepo.GetByUserID(ctx, userID)
}

func (s *accountService) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.Account, error) {
	return s.accountRepo.GetByUserIDAndProvider(ctx, userID, provider)
}

func (s *accountService) GetByProviderAndAccountID(ctx context.Context, provider, accountID string) (*models.Account, error) {
	return s.accountRepo.GetByProviderAndAccountID(ctx, provider, accountID)
}

func (s *accountService) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.BeforeUpdate != nil {
		if err := s.dbHooks.Accounts.BeforeUpdate(account); err != nil {
			return nil, err
		}
	}

	updatedAccount, err := s.accountRepo.Update(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.dbHooks != nil && s.dbHooks.Accounts != nil && s.dbHooks.Accounts.AfterUpdate != nil {
		if err := s.dbHooks.Accounts.AfterUpdate(*updatedAccount); err != nil {
			return nil, err
		}
	}

	return updatedAccount, nil
}

func (s *accountService) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	return s.accountRepo.UpdateFields(ctx, userID, fields)
}
